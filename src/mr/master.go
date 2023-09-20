package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type MasterTaskStatus int

const (
	Idle MasterTaskStatus = iota
	InProgress
	Completed
)

type State int

const (
	Map State = iota
	Reduce
	Exit
	Wait
)

type Master struct {
	TaskQueue     chan *Task          // 等待执行的task
	TaskMeta      map[int]*MasterTask // 当前所有task的信息
	MasterPhase   State               // MasterPhase: Map, Reduce, Exit, Wait
	NReduce       int
	InputFiles    []string
	Intermediates [][]string // Map任务产生的R个中间文件的信息
}

type MasterTask struct {
	TaskStatus    MasterTaskStatus
	StartTime     time.Time
	TaskReference *Task
}

type Task struct {
	Input         string
	TaskState     State
	NReducer      int
	TaskNumber    int
	Intermediates []string
	Output        string
}

var mu sync.Mutex

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
func (m *Master) Done() bool {
	mu.Lock()
	defer mu.Unlock()
	ret := m.MasterPhase == Exit
	return ret
}

// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{
		TaskQueue:     make(chan *Task, max(nReduce, len(files))),
		TaskMeta:      make(map[int]*MasterTask),
		MasterPhase:   Map,
		NReduce:       nReduce,
		InputFiles:    files,
		Intermediates: make([][]string, nReduce),
	}

	//切分文件
	//create Map Task
	m.createMapTask()

	m.server()

	// 10s检查一次超时
	go m.catchTimeOut()

	return &m
}

func (m *Master) catchTimeOut() {
	for{
		time.Sleep(time.Second * 10)
		mu.Lock()
		if m.MasterPhase == Exit {
			mu.Unlock()
			return
		}
		for _, masterTask := range m.TaskMeta {
			if masterTask.TaskStatus == InProgress && time.Now().Sub(masterTask.StartTime) > time.Second * 10 {
				m.TaskQueue <- masterTask.TaskReference
				masterTask.TaskStatus = Idle
			}
		}
		mu.Unlock()

	}
}

func (m *Master) createMapTask() {
	// 根据传入的filename，每个文件对应一个map task
	for idx, filename := range m.InputFiles {
		taskMeta := Task{
			Input:      filename,
			TaskState:  Map,
			NReducer:   m.NReduce,
			TaskNumber: idx,
		}
		m.TaskQueue <- &taskMeta
		m.TaskMeta[idx] = &MasterTask{
			TaskStatus:    Idle,
			TaskReference: &taskMeta,
		}
	}
}

func (m *Master) createReduceTask() {
	m.TaskMeta = make(map[int]*MasterTask)
	for idx, files := range m.Intermediates {
		taskMeta := Task{
			TaskState:  Reduce,
			NReducer:   m.NReduce,
			TaskNumber: idx,
			Intermediates: files,
		}
		m.TaskQueue <- &taskMeta
		m.TaskMeta[idx] = &MasterTask{
			TaskStatus:    Idle,
			TaskReference: &taskMeta,
		}
	}
}

func (m *Master) AssignTask(args *ExampleArgs, reply *Task) error {
	mu.Lock()
	defer mu.Unlock()
	if len(m.TaskQueue) > 0 {
		*reply = *<-m.TaskQueue
		// record the start time
		m.TaskMeta[reply.TaskNumber].StartTime = time.Now()
		m.TaskMeta[reply.TaskNumber].TaskStatus = InProgress
	} else if m.MasterPhase == Exit {
		*reply = Task{
			TaskState: Exit,
		}
	} else {
		*reply = Task{
			TaskState: Wait,
		}
	}

	return nil
}

func (m *Master) TaskCompleted(task *Task, reply *ExampleReply) error {
	//更新task状态
	mu.Lock()
	defer mu.Unlock()
	if task.TaskState != m.MasterPhase || m.TaskMeta[task.TaskNumber].TaskStatus == Completed {
		// 因为worker写在同一个文件磁盘上，对于重复的结果要丢弃
		return nil
	}
	m.TaskMeta[task.TaskNumber].TaskStatus = Completed
	go m.processTaskResult(task)
	return nil
}

func (m *Master) processTaskResult(task *Task) {
	mu.Lock()
	defer mu.Unlock()
	if task.TaskState == Map {
		for reduceTaskID, filePath := range task.Intermediates {
			m.Intermediates[reduceTaskID] = append(m.Intermediates[reduceTaskID], filePath)
		}
		if m.allTaskDone() {
			m.createReduceTask()
			m.MasterPhase = Reduce
		}
	} else if task.TaskState == Reduce {
		if m.allTaskDone() {

			// change to EXIT after get all task done
			m.MasterPhase = Exit
		}
	}
}

func (m *Master) allTaskDone() bool {
	for _, task := range m.TaskMeta {
		if task.TaskStatus != Completed {
			return false
		}
	}
	return true
}
