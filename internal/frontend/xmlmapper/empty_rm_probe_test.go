package xmlmapper

import (
	"fmt"
	"strings"
	"testing"
)

func TestEmptyResultMapProbe(t *testing.T) {
	xml := `<mapper namespace="ns">
	<resultMap id="Empty" type="User"></resultMap>
	<select id="s" resultMap="Empty">SELECT 1</select>
</mapper>`
	m, err := Parse(strings.NewReader(xml), "x.xml")
	fmt.Printf("mapper=%+v err=%v\n", m, err)
	if m != nil {
		for id, rm := range m.ResultMaps {
			fmt.Printf("resultMap %q: Fields=%v\n", id, rm.Fields)
		}
	}
}
