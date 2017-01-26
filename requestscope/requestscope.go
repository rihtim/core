package requestscope

type RequestScope struct {
	data map[string]interface{}
}

func Init() RequestScope {
	requestScope := RequestScope{
		data:make(map[string]interface{}),
	}
	return requestScope
}

func (rs RequestScope) Set(key string, value interface{}) {
	rs.data[key] = value
}

func (rs RequestScope) Get(key string) interface{} {
	return rs.data[key]
}

func (rs RequestScope) Contains(key string) bool {
	_, contains := rs.data[key]
	return contains
	//return false
}

func (rs RequestScope) IsEmpty() bool {
	return rs.data == nil || len(rs.data) != 0
}