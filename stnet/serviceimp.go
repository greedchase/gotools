package stnet

type ServiceImp interface {
	Init() bool
	Loop()
	Destroy()

	//in hash thread by HashProcessor
	HandleMessage(current *CurrentContent, msgID uint64, msg interface{})
	//in main thread of service
	HandleError(*CurrentContent, error)
	SessionClose(sess *Session)
	HeartBeatTimeOut(sess *Session)
	//in random thread
	SessionOpen(sess *Session)

	// Unmarshal protocol parsed
	//lenParsed is the length read from 'data'.
	//msgID and msg are messages parsed from data.
	//when lenParsed <= 0 or msgID < 0,msg and err will be ignored.
	Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error)

	// HashProcessor sess msgID msg are returned by func of Unmarshal
	//processorID is the thread who process this msg;it should be between 1-ProcessorThreadsNum.
	//if processorID == 0, it only uses main thread of the service.
	//if processorID < 0, it will use hash of session id.
	HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int)
}

type LoopService interface {
	Init() bool
	Loop()
}

type JsonProto struct {
	CmdId   uint64 `json:"id"`
	CmdData []byte `json:"cmd"`
}
type JsonService interface {
	Init() bool
	Loop()
	Handle(current *CurrentContent, cmd JsonProto, e error)
	HashProcessor(current *CurrentContent, cmd JsonProto) (processorID int)
}

type SpbService interface {
	Init() bool
	Loop()
	Handle(current *CurrentContent, cmdId uint64, cmd interface{}, e error)
	HashProcessor(current *CurrentContent, cmdId uint64) (processorID int)
}
