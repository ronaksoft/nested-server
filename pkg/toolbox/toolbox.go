package tools

/*
   Creation Time: 2018 - Apr - 07
   Created by:  Ehsan N. Moosa (ehsan)
   Maintainers:
       1.  Ehsan N. Moosa (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/


type (
	M  map[string]interface{}
	MS map[string]string
	MI map[string]int64
	MB map[string]bool
)


func (m M) KeysToArray() []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	return arr
}

func (m M) ValuesToArray() []interface{} {
	arr := make([]interface{}, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func (m MB) AddKeys(keys ...[]string) {
	for _, arr := range keys {
		for _, key := range arr {
			m[key] = true
		}
	}
}

func (m MB) KeysToArray() []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	return arr
}
