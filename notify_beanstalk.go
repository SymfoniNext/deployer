package main

import (
	"bytes"
	"log"
	"text/template"
	"time"

	"github.com/kr/beanstalk"
)

type NotifyBeanstalkd struct {
	addr string
	tube string
	t    *template.Template

	Deadline time.Duration
	Priority uint32
	Delay    time.Duration
}

func NewNotifyBeanstalkd(addr, tube, body_template string) NotifyBeanstalkd {
	return NotifyBeanstalkd{
		addr: addr,
		tube: tube,
		t:    template.Must(template.New("body").Parse(body_template)),

		Deadline: time.Minute,
		Priority: 10,
		Delay:    0,
	}
}

func (n NotifyBeanstalkd) Notify(p *Payload) {

	body := n.parseBody(p)
	if len(body) == 0 {
		log.Println("Notify body is empty!")
		return
	}

	bs, err := beanstalk.Dial("tcp", n.addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer bs.Close()

	tube := &beanstalk.Tube{bs, n.tube}
	job_id, err := tube.Put(body, n.Priority, n.Delay, n.Deadline)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Added job with id %d\n", job_id)
}

func (n NotifyBeanstalkd) parseBody(p *Payload) []byte {

	var buf bytes.Buffer
	if err := n.t.Execute(&buf, p); err != nil {
		log.Println(err)
	}
	return buf.Bytes()
}
