package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"


type Master struct {
	// Your definitions here.

}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}


//
// start a thread that listens for RPCs from worker.go
//
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

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false

	// Your code here.


	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}
	TaskQueue:		make(chan *Task, max(nReduce, len(files))),
	TaskMeta:		make(map[int]*MasterTask),					//保留task引用和状态
	MasterPhase:	Map,										// MasterPhase: Map, Reduce,表明当前状态
	NReduce:		nReduce,									// Reduce任务数
	InputFiles:		files,										// 输入文件
	Intermediates:	make([][]string, nReduce),					// 中间结果
	


	m.server()
	return &m
}
