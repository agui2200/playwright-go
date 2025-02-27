package playwright

type workerImpl struct {
	channelOwner
	page    *pageImpl
	context *browserContextImpl
}

func (w *workerImpl) URL() string {
	return w.initializer["url"].(string)
}

func (w *workerImpl) Evaluate(expression string, options ...interface{}) (interface{}, error) {
	var arg interface{}
	forceExpression := false
	if !isFunctionBody(expression) {
		forceExpression = true
	}
	if len(options) == 1 {
		arg = options[0]
	} else if len(options) == 2 {
		arg = options[0]
		forceExpression = options[1].(bool)
	}
	result, err := w.channel.Send("evaluateExpression", map[string]interface{}{
		"expression": expression,
		"isFunction": !forceExpression,
		"arg":        serializeArgument(arg),
	})
	if err != nil {
		return nil, err
	}
	return parseResult(result), nil
}

func (w *workerImpl) EvaluateHandle(expression string, options ...interface{}) (JSHandle, error) {
	var arg interface{}
	forceExpression := false
	if !isFunctionBody(expression) {
		forceExpression = true
	}
	if len(options) == 1 {
		arg = options[0]
	} else if len(options) == 2 {
		arg = options[0]
		forceExpression = options[1].(bool)
	}
	result, err := w.channel.Send("evaluateExpressionHandle", map[string]interface{}{
		"expression": expression,
		"isFunction": !forceExpression,
		"arg":        serializeArgument(arg),
	})
	if err != nil {
		return nil, err
	}
	return fromChannel(result).(*jsHandleImpl), nil
}

func (w *workerImpl) onClose() {
	if w.page != nil {
		w.page.Lock()
		workers := make([]Worker, 0)
		for i := 0; i < len(w.page.workers); i++ {
			if w.page.workers[i] != w {
				workers = append(workers, w.page.workers[i])
			}
		}
		w.page.workers = workers
		w.page.Unlock()
	}
	if w.context != nil {
		w.context.Lock()
		workers := make([]*workerImpl, 0)
		for i := 0; i < len(w.context.serviceWorkers); i++ {
			if w.context.serviceWorkers[i] != w {
				workers = append(workers, w.context.serviceWorkers[i])
			}
		}
		w.context.serviceWorkers = workers
		w.context.Unlock()
	}
	w.Emit("close", w)
}

func (w *workerImpl) WaitForEvent(event string, predicate ...interface{}) interface{} {
	return <-waitForEvent(w, event, w.context.timeoutSettings.Timeout(), predicate...)
}

func (w *workerImpl) ExpectEvent(event string, cb func() error, predicates ...interface{}) (interface{}, error) {
	args := []interface{}{event}
	if len(predicates) == 1 {
		args = append(args, predicates[0])
	}
	return newExpectWrapper(w.WaitForEvent, args, cb)
}

func newWorker(parent *channelOwner, objectType string, guid string, initializer map[string]interface{}) *workerImpl {
	bt := &workerImpl{}
	bt.createChannelOwner(bt, parent, objectType, guid, initializer)
	bt.channel.On("close", bt.onClose)
	return bt
}
