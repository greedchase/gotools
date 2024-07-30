package stnet

import (
	"fmt"
	"math/rand"
	"reflect"
)

// ServiceImpBase
type ServiceBase struct {
}

func (service *ServiceBase) Init() bool {
	return true
}
func (service *ServiceBase) Loop() {

}
func (service *ServiceBase) Destroy() {

}
func (service *ServiceBase) HandleMessage(current *CurrentContent, msgID uint64, msg interface{}) {

}
func (service *ServiceBase) SessionOpen(sess *Session) {

}
func (service *ServiceBase) SessionClose(sess *Session) {

}
func (service *ServiceBase) HeartBeatTimeOut(sess *Session) {
	sess.Close()
}
func (service *ServiceBase) HandleError(current *CurrentContent, err error) {
	sysLog.Error(err.Error())
	current.Sess.Close()
}
func (service *ServiceBase) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	return len(data), -1, nil, nil
}

// processorID is the thread who process this msg;it should between 1-ProcessorThreadsNum.
// if processorID == 0, it only uses main thread of the service.
// if processorID < 0, it will use hash of session id.
func (service *ServiceBase) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	return -1
}

// ServiceImpEcho
type ServiceEcho struct {
	ServiceBase
}

func (service *ServiceEcho) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	sess.Send(data, sess.peer)
	return len(data), -1, nil, nil
}

func (service *ServiceEcho) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	return int(current.Sess.GetID())
}

type ServiceLoop struct {
	ServiceBase
	imp LoopService
}

func (service *ServiceLoop) Init() bool {
	return service.imp.Init()
}

func (service *ServiceLoop) Loop() {
	service.imp.Loop()
}

type ServiceProxyS struct {
	ServiceBase
	remote   *Service
	remoteip []string
	weight   []int
}

func (service *ServiceProxyS) SessionOpen(sess *Session) {
	rip := service.remoteip[0]
	ln := len(service.weight)
	if ln > 1 {
		r := rand.Int() % service.weight[ln-1]
		for i := 0; i < ln; i++ {
			if r < service.weight[i] {
				rip = service.remoteip[i]
				break
			}
		}
	}
	sess.UserData = service.remote.NewConnect(rip, sess)
}
func (service *ServiceProxyS) SessionClose(sess *Session) {
	if sess.UserData != nil {
		sess.UserData.(*Connect).Close()
	}
}
func (service *ServiceProxyS) HeartBeatTimeOut(sess *Session) {
	sess.Close()
}
func (service *ServiceProxyS) HandleError(current *CurrentContent, err error) {
	current.Sess.Close()
}
func (service *ServiceProxyS) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	sess.UserData.(*Connect).Send(data)
	return len(data), -1, nil, nil
}
func (service *ServiceProxyS) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	return int(current.Sess.GetID())
}

type ServiceProxyC struct {
	ServiceBase
}

func (service *ServiceProxyC) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	sess.UserData.(*Session).Send(data, sess.peer)
	return len(data), -1, nil, nil
}
func (service *ServiceProxyC) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	return int(current.Sess.GetID())
}

// ServiceSpb
type ServiceSpb struct {
	ServiceBase
	imp    SpbService
	msgReg map[uint64]reflect.Type
}

func NewServiceSpb(imp SpbService) *ServiceSpb {
	return &ServiceSpb{ServiceBase{}, imp, make(map[uint64]reflect.Type)}
}

func (service *ServiceSpb) RegisterMsg(msgId uint64, msg interface{}) error {
	t := reflect.TypeOf(msg)
	if msg == nil || t.Kind() == reflect.Ptr {
		return fmt.Errorf("type of msg cannot be ptr or nil")
	}
	service.msgReg[msgId] = t
	return nil
}

func (service *ServiceSpb) Init() bool {
	return service.imp.Init()
}

func (service *ServiceSpb) Loop() {
	service.imp.Loop()
}

func (service *ServiceSpb) HandleMessage(current *CurrentContent, msgID uint64, msg interface{}) {
	t, ok := service.msgReg[msgID]
	if !ok {
		service.imp.Handle(current, msgID, msg, nil)
		return
	}

	var d []byte
	if msg != nil {
		d, ok = msg.([]byte)
		if !ok {
			service.imp.Handle(current, msgID, msg, fmt.Errorf("msg unmarshal failed id=%d,err=msg not marshaled bytes", msgID))
			return
		}
	}
	m := reflect.New(t).Interface()
	e := Unmarshal(d, m, EncodeTyepSpb)
	if e != nil {
		service.imp.Handle(current, msgID, nil, fmt.Errorf("msg unmarshal failed id=%d,err=%s", msgID, e.Error()))
	} else {
		service.imp.Handle(current, msgID, m, nil)
	}
}
func (service *ServiceSpb) HandleError(current *CurrentContent, err error) {
	service.imp.Handle(current, 0, nil, err)
}
func (service *ServiceSpb) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	if len(data) < 4 {
		return 0, 0, nil, nil
	}
	msgLen := MsgLen(data)
	if msgLen < 4 || msgLen >= uint32(MaxMsgSize) {
		return len(data), 0, nil, fmt.Errorf("message length is invalid: %d", msgLen)
	}

	if len(data) < int(msgLen) {
		return 0, 0, nil, nil
	}
	cmd := JsonProto{}
	e := Unmarshal(data[4:msgLen], &cmd, EncodeTyepSpb)
	if e != nil {
		return int(msgLen), 0, nil, e
	}
	return int(msgLen), int64(cmd.CmdId), cmd.CmdData, nil
}
func (service *ServiceSpb) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	return service.imp.HashProcessor(current, msgID)
}

func SendSpbCmd(sess *Session, msgID uint64, msg interface{}) error {
	d, e := Marshal(msg, EncodeTyepSpb)
	if e != nil {
		return e
	}
	cmd := JsonProto{msgID, d}
	buf, e := EncodeProtocol(cmd, EncodeTyepSpb)
	if e != nil {
		return e
	}
	return sess.Send(buf, nil)
}

// ServiceJson
type ServiceJson struct {
	ServiceBase
	imp JsonService
}

func (service *ServiceJson) Init() bool {
	return service.imp.Init()
}

func (service *ServiceJson) Loop() {
	service.imp.Loop()
}

func (service *ServiceJson) HandleMessage(current *CurrentContent, msgID uint64, msg interface{}) {
	var d []byte
	var ok bool
	if msg != nil {
		d, ok = msg.([]byte)
		if !ok {
			service.imp.Handle(current, JsonProto{msgID, nil}, fmt.Errorf("format of msg should be []byte, id=%d", msgID))
			return
		}
	}
	service.imp.Handle(current, JsonProto{msgID, d}, nil)
}
func (service *ServiceJson) HandleError(current *CurrentContent, err error) {
	service.imp.Handle(current, JsonProto{}, err)
}
func (service *ServiceJson) Unmarshal(sess *Session, data []byte) (lenParsed int, msgID int64, msg interface{}, err error) {
	if len(data) < 4 {
		return 0, 0, nil, nil
	}
	msgLen := MsgLen(data)
	if msgLen < 4 || msgLen >= uint32(MaxMsgSize) {
		return len(data), 0, nil, fmt.Errorf("message length is invalid: %d", msgLen)
	}

	if len(data) < int(msgLen) {
		return 0, 0, nil, nil
	}
	cmd := JsonProto{}
	e := Unmarshal(data[4:msgLen], &cmd, EncodeTyepJson)
	if e != nil {
		return int(msgLen), 0, nil, e
	}
	return int(msgLen), int64(cmd.CmdId), cmd.CmdData, nil
}
func (service *ServiceJson) HashProcessor(current *CurrentContent, msgID uint64, msg interface{}) (processorID int) {
	var d []byte
	if msg != nil {
		d = msg.([]byte)
	}
	return service.imp.HashProcessor(current, JsonProto{msgID, d})
}

func SendJsonCmd(sess *Session, msgID uint64, msg []byte) error {
	cmd := JsonProto{msgID, msg}
	buf, e := EncodeProtocol(cmd, EncodeTyepJson)
	if e != nil {
		return e
	}
	return sess.Send(buf, nil)
}
