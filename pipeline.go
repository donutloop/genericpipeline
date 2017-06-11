package genericpipeline

import (
	"context"
	"fmt"
	"reflect"
)

// Execute is function that represent a function for the pipeline
type Func interface{}

type Pipeline struct {
	kindOfFirstInput reflect.Kind
	funcs            []Func
	cw               *connectorenWrapper
	errChan          chan error
}

// New builds pipeline
// The parameter "Input" is a inital value for the pipeline.
// The parameters "command" are commands for the pipline
func Create(funcs ...Func) (*Pipeline, error) {
	p := new(Pipeline)
	p.ValidateFuncs(funcs)
	p.funcs = append(make([]Func, 0, len(funcs)), funcs...)
	if err := p.Setup(context.TODO()); err != nil {
		return p, err
	}
	return p, nil
}

func CreateWithContext(ctx context.Context, funcs ...Func) (*Pipeline, error) {
	p := new(Pipeline)
	p.ValidateFuncs(funcs)
	p.funcs = append(make([]Func, 0, len(funcs)), funcs...)
	if err := p.Setup(ctx); err != nil {
		return p, err
	}
	return p, nil
}

type connectorenWrapper struct {
	Connectoren []chan interface{}
}

func (cw *connectorenWrapper) getFirstConnector() chan interface{} {
	return cw.Connectoren[0]
}

func (cw *connectorenWrapper) getLastConnector() chan interface{} {
	return cw.Connectoren[len(cw.Connectoren)-1]
}

func (p *Pipeline) ValidateFuncs(funcs []Func) {
	firstFunc := true
	var outType reflect.Kind
	for _, f := range funcs {
		fnType := reflect.TypeOf(f)

		// panic if conditions not met (because it's a programming error to have that happen)
		switch {
		case fnType.Kind() != reflect.Func:
			panic("value must be a function")
		case fnType.NumIn() != 1:
			panic("value must take exactly one input argument")
		case fnType.NumOut() != 2:
			panic("value must take exactly one input argument")
		}

		if firstFunc {
			p.kindOfFirstInput = fnType.In(0).Kind()
			outType = fnType.Out(0).Kind()
			firstFunc = false
			continue
		}

		if fnType.In(0).Kind() != outType {
			panic("todo")
		}

		outType = fnType.Out(0).Kind()
	}
}

func (p *Pipeline) Setup(ctx context.Context) error {

	if len(p.funcs) < 2 {
		return fmt.Errorf("Added to less funcs (Count: %d)", len(p.funcs))
	}

	p.errChan = make(chan error)
	p.cw = p.createConnectors(len(p.funcs) + 1)
	for index, f := range p.funcs {
		param := funcWrapperParam{
			funky:  f,
			input:  p.cw.Connectoren[index],
			output: p.cw.Connectoren[index+1],
			err:    p.errChan,
			ctx:    ctx,
		}

		go p.convertToFuncWrapper(param)()
	}

	return nil
}

func (p *Pipeline) Output() (interface{}, error) {

	var err error
	var output interface{}

	select {
	case err = <-p.errChan:
		return nil, err
	case output = <-p.cw.getLastConnector():
		return output, nil
	}

	return nil, nil
}

func (p *Pipeline) Input(v interface{}) {

	if p.kindOfFirstInput != reflect.TypeOf(reflect.ValueOf(v).Interface()).Kind() {
		panic("value must be exactly the same kind")
	}

	connector := p.cw.getFirstConnector()
	connector <- v
}

func (p *Pipeline) createConnectors(funcCount int) *connectorenWrapper {

	cw := &connectorenWrapper{
		Connectoren: make([]chan interface{}, funcCount),
	}

	for index := 0; index < funcCount; index++ {
		cw.Connectoren[index] = make(chan interface{})
	}

	return cw
}

func (p *Pipeline) closeConnectors(connectoren []chan interface{}) {
	for _, connector := range connectoren {
		close(connector)
	}
}

type funcWrapperParam struct {
	funky  Func
	input  chan interface{}
	output chan interface{}
	err    chan error
	ctx    context.Context
}

func (p *Pipeline) convertToFuncWrapper(param funcWrapperParam) func() {
	return func() {
		input := <-param.input

		select {
		case <-param.ctx.Done():
			param.err <- param.ctx.Err()
			return
		default:

		}

		inputVals := []reflect.Value{reflect.ValueOf(input)}
		outputVals := reflect.ValueOf(param.funky).Call(inputVals)
		data := outputVals[0].Interface()
		err, ok := outputVals[1].Interface().(error)
		if ok && err != nil {
			param.err <- err
		} else {
			param.output <- data
		}
	}
}
