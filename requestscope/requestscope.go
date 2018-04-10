package requestscope

import (
	"strconv"
	"github.com/rihtim/core/log"
)

type RequestScope struct {
	data map[string]interface{}
}

func Init() RequestScope {
	requestScope := RequestScope{
		data: make(map[string]interface{}),
	}
	return requestScope
}

func (rs RequestScope) Copy() RequestScope {
	clone := RequestScope{data: map[string]interface{}{}}
	for k, v := range rs.data {
		clone.data[k] = v
	}
	return clone
}

func (rs RequestScope) Set(key string, value interface{}) {
	log.Debug("RequestScope.Set:", key, value)
	rs.data[key] = value
}

func (rs RequestScope) Get(key string) interface{} {
	log.Debug("RequestScope.Get:", key)
	return rs.data[key]
}

func (rs RequestScope) Delete(key string) {
	log.Debug("RequestScope.Delete:", key)
	delete(rs.data, key)
}

func (rs RequestScope) Contains(key string) bool {
	_, contains := rs.data[key]
	log.Debug("RequestScope.Has:" + key + " = " + strconv.FormatBool(contains))
	return contains
}

func (rs RequestScope) IsEmpty() bool {
	return rs.data == nil || len(rs.data) == 0
}
