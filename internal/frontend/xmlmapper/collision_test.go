package xmlmapper

import (
	"strings"
	"testing"
)

func TestCollisionSilentOverwrite(t *testing.T) {
	xml := `
<mapper namespace="ns">
  <sql id="ns.a">NSA_BODY</sql>
  <sql id="a">A_BODY</sql>
  <select id="s1"><include refid="ns.a"/></select>
</mapper>`
	_, err := Parse(strings.NewReader(xml), "x.xml")
	if err == nil || !strings.Contains(err.Error(), `duplicate sql fragment "ns.a"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestCircularAliasBypass(t *testing.T) {
	// A includes B via namespaced key; B includes A via namespaced key
	xml := `
<mapper namespace="ns">
  <sql id="a">P <include refid="ns.b"/></sql>
  <sql id="b">Q <include refid="ns.a"/></sql>
  <select id="s1"><include refid="a"/></select>
</mapper>`
	_, err := Parse(strings.NewReader(xml), "x.xml")
	if err == nil || !strings.Contains(err.Error(), `circular SQL include "ns.a"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectsSiblingPropertyRef(t *testing.T) {
	xml := `
<mapper namespace="ns">
  <sql id="frag">SELECT ${col2}</sql>
  <select id="s1">
    <include refid="frag">
      <property name="col" value="x"/>
      <property name="col2" value="${col}_y"/>
    </include>
  </select>
</mapper>`
	_, err := Parse(strings.NewReader(xml), "x.xml")
	if err == nil || !strings.Contains(
		err.Error(),
		`include property "col2" contains unresolved placeholder "${col}"`,
	) {
		t.Fatalf("error = %v", err)
	}
}
