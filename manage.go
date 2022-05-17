package channels

import (
	"fmt"

	"github.com/google/uuid"
)

type ChannelInfo_t struct {
	id uint32
	C  chan interface{}
}

type notifyInfo_t struct {
	t    notifyType_t
	info ChannelInfo_t
}

type notifyType_t int

const (
	add notifyType_t = iota
	remove
	release
)

type IntegratedChannel_t struct {
	subInfo     []ChannelInfo_t
	C           chan interface{}
	notify      chan notifyInfo_t
	response    chan interface{}
	id          uint32
	channelSize int
}

type ChannelType int

const (
	ChannelBroadCast ChannelType = iota
	ChannelCollect
)

func CreateIntegratedCh(size int, chType ChannelType) (*IntegratedChannel_t, error) {
	if size == 0 {
		return nil, fmt.Errorf("Incorrect size\n")
	}

	c := &IntegratedChannel_t{
		C:           make(chan interface{}, size),
		subInfo:     []ChannelInfo_t{},
		channelSize: size,
		id:          uuid.Must(uuid.NewRandom()).ID(),
		notify:      make(chan notifyInfo_t),
		response:    make(chan interface{}),
	}

	switch chType {
	case ChannelBroadCast:
		go broadCast(c)
		break
	case ChannelCollect:
		go collect(c)
		break
	}

	return c, nil
}

func DeleteIntegratedCh(in *IntegratedChannel_t) {
	in.notify <- notifyInfo_t{
		t:    release,
		info: ChannelInfo_t{},
	}
	<-in.response
	close(in.response)
}

func (in *IntegratedChannel_t) RegisterCh(c chan interface{}) ChannelInfo_t {
	id := uuid.Must(uuid.NewRandom())
	info := ChannelInfo_t{
		id: id.ID(),
		C:  c,
	}

	in.notify <- notifyInfo_t{
		t:    add,
		info: info,
	}
	<-in.response
	return info
}

func (in *IntegratedChannel_t) GetCh() ChannelInfo_t {
	id := uuid.Must(uuid.NewRandom())
	info := ChannelInfo_t{
		id: id.ID(),
		C:  make(chan interface{}, in.channelSize),
	}

	in.notify <- notifyInfo_t{
		t:    add,
		info: info,
	}
	<-in.response
	return info
}

func (in *IntegratedChannel_t) RemoveCh(info ChannelInfo_t) {
	in.notify <- notifyInfo_t{
		t:    remove,
		info: info,
	}

	<-in.response
	return
}

func broadCast(in *IntegratedChannel_t) {
	for {
		select {
		case d := <-in.C:
			for _, c := range in.subInfo {
				c.C <- d
			}
		case d := <-in.notify:
			switch d.t {
			case add:
				in.subInfo = append(in.subInfo, d.info)
			case remove:
				for i, v := range in.subInfo {
					if v.id == d.info.id {
						close(v.C)
						in.subInfo = append(in.subInfo[:i], in.subInfo[i+1:]...)
						break
					}
				}
			case release:
				for _, v := range in.subInfo {
					close(v.C)
				}
				close(in.C)
				close(in.notify)
				in.response <- "done"
				return
			}
			in.response <- "done"
		}
	}
}

func collect(in *IntegratedChannel_t) {
	for {
		select {
		case d, ok := <-in.notify:
			if !ok {
				return
			}
			switch d.t {
			case add:
				in.subInfo = append(in.subInfo, d.info)
				go func(c chan interface{}) {
					for {
						select {
						case d, ok := <-c:
							if !ok {
								return
							}
							in.C <- d
						}
					}
				}(d.info.C)
			case remove:
				for i, v := range in.subInfo {
					if v.id == d.info.id {
						close(v.C)
						in.subInfo = append(in.subInfo[:i], in.subInfo[i+1:]...)
						break
					}
				}
			case release:
				fmt.Println("Release")
				for _, v := range in.subInfo {
					close(v.C)
				}
				close(in.C)
				close(in.notify)
				in.response <- "done"
				return
			}
			in.response <- "done"
		}
	}
}
