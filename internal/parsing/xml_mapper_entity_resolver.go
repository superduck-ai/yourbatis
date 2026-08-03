package parsing

import (
	"bytes"
	_ "embed"
	"strings"
)

const (
	IBATIS_CONFIG_SYSTEM  = "ibatis-3-config.dtd"
	IBATIS_MAPPER_SYSTEM  = "ibatis-3-mapper.dtd"
	MYBATIS_CONFIG_SYSTEM = "mybatis-3-config.dtd"
	MYBATIS_MAPPER_SYSTEM = "mybatis-3-mapper.dtd"

	MYBATIS_CONFIG_DTD = "dtd/mybatis-3-config.dtd"
	MYBATIS_MAPPER_DTD = "dtd/mybatis-3-mapper.dtd"
)

type XMLMapperEntityResolver struct{}

//go:embed dtd/mybatis-3-config.dtd
var mybatisConfigDTD []byte

//go:embed dtd/mybatis-3-mapper.dtd
var mybatisMapperDTD []byte

func (resolver *XMLMapperEntityResolver) ResolveEntity(publicID, systemID string) ([]byte, error) {
	_ = publicID
	if systemID == "" {
		return nil, nil
	}

	lowerCaseSystemID := strings.ToLower(systemID)
	switch {
	case strings.Contains(lowerCaseSystemID, MYBATIS_CONFIG_SYSTEM),
		strings.Contains(lowerCaseSystemID, IBATIS_CONFIG_SYSTEM):
		return bytes.Clone(mybatisConfigDTD), nil
	case strings.Contains(lowerCaseSystemID, MYBATIS_MAPPER_SYSTEM),
		strings.Contains(lowerCaseSystemID, IBATIS_MAPPER_SYSTEM):
		return bytes.Clone(mybatisMapperDTD), nil
	default:
		return nil, nil
	}
}
