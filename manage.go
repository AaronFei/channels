package channels

import (
	"fmt"

	"github.com/google/uuid"
)

type ChannelInfo_t[T any] struct {
	id uint32
	C  chan T
}

type notifyInfo_t[T any] struct {
	t    notifyType_t
	info ChannelInfo_t[T]
}

type notifyType_t int

const (
	add notifyType_t = iota
	remove
	release
)

type IntegratedChannel_t[T any] struct {
	subInfo     map[uint32]ChannelInfo_t[T]
	C           chan T
	notify      chan notifyInfo_t[T]
	response    chan error
	channelSize int
}

func CreateCollectedCh[T any](size int) (*IntegratedChannel_t[T], error) {
	if size == 0 {
		return nil, fmt.Errorf("Incorrect size\n")
	}

	c := &IntegratedChannel_t[T]{
		C:           make(chan T, size),
		subInfo:     map[uint32]ChannelInfo_t[T]{},
		channelSize: size,
		notify:      make(chan notifyInfo_t[T]),
		response:    make(chan error),
	}

	go collect(c)

	return c, nil
}

func CreateBroadcastCh[T any](size int) (*IntegratedChannel_t[T], error) {
	if size == 0 {
		return nil, fmt.Errorf("Incorrect size\n")
	}

	c := &IntegratedChannel_t[T]{
		C:           make(chan T, size),
		subInfo:     map[uint32]ChannelInfo_t[T]{},
		channelSize: size,
		notify:      make(chan notifyInfo_t[T]),
		response:    make(chan error),
	}

	go broadCast(c)

	return c, nil
}

func (in *IntegratedChannel_t[T]) ReleaseChannels() (err error) {
	in.notify <- notifyInfo_t[T]{
		t:    release,
		info: ChannelInfo_t[T]{},
	}

	err = <-in.response
	close(in.response)
	return
}

func (in *IntegratedChannel_t[T]) RegisterCh(c chan T) (ChannelInfo_t[T], error) {
	id := uuid.Must(uuid.NewRandom())
	info := ChannelInfo_t[T]{
		id: id.ID(),
		C:  c,
	}

	in.notify <- notifyInfo_t[T]{
		t:    add,
		info: info,
	}

	return info, <-in.response
}

func (in *IntegratedChannel_t[T]) RemoveCh(info ChannelInfo_t[T]) error {
	in.notify <- notifyInfo_t[T]{
		t:    remove,
		info: info,
	}

	return <-in.response
}

func broadCast[T any](in *IntegratedChannel_t[T]) {
	for {
		select {
		case d := <-in.C:
			for _, c := range in.subInfo {
				c.C <- d
			}
		case d, ok := <-in.notify:
			if !ok {
				in.response <- fmt.Errorf("Notify ch invalid")
				return
			}
			switch d.t {
			case add:
				in.subInfo[d.info.id] = d.info
			case remove:
				delete(in.subInfo, d.info.id)
			case release:
				close(in.C)
				close(in.notify)
				in.response <- nil
				return
			}
			in.response <- nil
		}
	}
}

func collect[T any](in *IntegratedChannel_t[T]) {
	for {
		select {
		case d, ok := <-in.notify:
			if !ok {
				in.response <- fmt.Errorf("Notify ch invalid")
				return
			}
			switch d.t {
			case add:
				in.subInfo[d.info.id] = d.info
				go func(c chan T) {
					for {
						select {
						case d, ok := <-c:
							if ok {
								in.C <- d
							} else {
								return
							}
						}
					}
				}(d.info.C)
			case remove:
				delete(in.subInfo, d.info.id)
			case release:
				close(in.C)
				close(in.notify)
				in.response <- nil
				return
			}
			in.response <- nil
		}
	}
}
