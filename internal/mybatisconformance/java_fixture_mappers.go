package mybatisconformance

import (
	"context"
	"strconv"

	yourbatis "github.com/superduck-ai/yourbatis"
)

// Each portable Java test owns one XML fixture and one generated mapper. This
// keeps the migration matrix honest: a case is only considered migrated when
// its XML passes through sqlmapgen and its generated builder is asserted.

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD01Mapper -sql ./javafixtures/d01.xml -out ./java_d01.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD02Mapper -sql ./javafixtures/d02.xml -out ./java_d02.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD03Mapper -sql ./javafixtures/d03.xml -out ./java_d03.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD04Mapper -sql ./javafixtures/d04.xml -out ./java_d04.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD05Mapper -sql ./javafixtures/d05.xml -out ./java_d05.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD06Mapper -sql ./javafixtures/d06.xml -out ./java_d06.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD07Mapper -sql ./javafixtures/d07.xml -out ./java_d07.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD08Mapper -sql ./javafixtures/d08.xml -out ./java_d08.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD09Mapper -sql ./javafixtures/d09.xml -out ./java_d09.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD10Mapper -sql ./javafixtures/d10.xml -out ./java_d10.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD11Mapper -sql ./javafixtures/d11.xml -out ./java_d11.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD12Mapper -sql ./javafixtures/d12.xml -out ./java_d12.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD13Mapper -sql ./javafixtures/d13.xml -out ./java_d13.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD14Mapper -sql ./javafixtures/d14.xml -out ./java_d14.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD15Mapper -sql ./javafixtures/d15.xml -out ./java_d15.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD16Mapper -sql ./javafixtures/d16.xml -out ./java_d16.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD17Mapper -sql ./javafixtures/d17.xml -out ./java_d17.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD18Mapper -sql ./javafixtures/d18.xml -out ./java_d18.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD19Mapper -sql ./javafixtures/d19.xml -out ./java_d19.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD20Mapper -sql ./javafixtures/d20.xml -out ./java_d20.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD21Mapper -sql ./javafixtures/d21.xml -out ./java_d21.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD23Mapper -sql ./javafixtures/d23.xml -out ./java_d23.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD24Mapper -sql ./javafixtures/d24.xml -out ./java_d24.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD25Mapper -sql ./javafixtures/d25.xml -out ./java_d25.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaD27Mapper -sql ./javafixtures/d27.xml -out ./java_d27.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaP01Mapper -sql ./javafixtures/p01.xml -out ./java_p01.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaP06Mapper -sql ./javafixtures/p06.xml -out ./java_p06.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaP07Mapper -sql ./javafixtures/p07.xml -out ./java_p07.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaP12Mapper -sql ./javafixtures/p12.xml -out ./java_p12.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaX01Mapper -sql ./javafixtures/x01.xml -out ./java_x01.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE01Mapper -sql ./javafixtures/e01.xml -out ./java_e01.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE02Mapper -sql ./javafixtures/e02.xml -out ./java_e02.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE03Mapper -sql ./javafixtures/e03.xml -out ./java_e03.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE04Mapper -sql ./javafixtures/e04.xml -out ./java_e04.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE05Mapper -sql ./javafixtures/e05.xml -out ./java_e05.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE06Mapper -sql ./javafixtures/e06.xml -out ./java_e06.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE07Mapper -sql ./javafixtures/e07.xml -out ./java_e07.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE08Mapper -sql ./javafixtures/e08.xml -out ./java_e08.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE09Mapper -sql ./javafixtures/e09.xml -out ./java_e09.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE10Mapper -sql ./javafixtures/e10.xml -out ./java_e10.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE11Mapper -sql ./javafixtures/e11.xml -out ./java_e11.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE12Mapper -sql ./javafixtures/e12.xml -out ./java_e12.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE13Mapper -sql ./javafixtures/e13.xml -out ./java_e13.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE14Mapper -sql ./javafixtures/e14.xml -out ./java_e14.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE15Mapper -sql ./javafixtures/e15.xml -out ./java_e15.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE16Mapper -sql ./javafixtures/e16.xml -out ./java_e16.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaO02Mapper -sql ./javafixtures/o02.xml -out ./java_o02.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE17Mapper -sql ./javafixtures/e17.xml -out ./java_e17.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE18Mapper -sql ./javafixtures/e18.xml -out ./java_e18.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE19Mapper -sql ./javafixtures/e19.xml -out ./java_e19.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE20Mapper -sql ./javafixtures/e20.xml -out ./java_e20.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE21Mapper -sql ./javafixtures/e21.xml -out ./java_e21.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE22Mapper -sql ./javafixtures/e22.xml -out ./java_e22.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE23Mapper -sql ./javafixtures/e23.xml -out ./java_e23.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE24Mapper -sql ./javafixtures/e24.xml -out ./java_e24.sqlmap.gen.go
//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper JavaE25Mapper -sql ./javafixtures/e25.xml -out ./java_e25.sqlmap.gen.go

type JavaD01Mapper interface {
	Run(ctx context.Context) error
}

type JavaD02Mapper interface {
	Run(ctx context.Context) error
}

type JavaD03Mapper interface {
	Run(ctx context.Context) error
}

type JavaD04Mapper interface {
	Run(ctx context.Context) error
}

type JavaD05Mapper interface {
	Run(ctx context.Context) error
}

type JavaD06Mapper interface {
	Run(ctx context.Context) error
}

type JavaD07Mapper interface {
	Run(ctx context.Context) error
}

type JavaD08Mapper interface {
	Run(ctx context.Context) error
}

type JavaD09Mapper interface {
	Run(ctx context.Context) error
}

type JavaD10Mapper interface {
	Run(ctx context.Context) error
}

type JavaD11Mapper interface {
	Run(ctx context.Context) error
}

type JavaD12Mapper interface {
	Run(ctx context.Context) error
}

type JavaD13Mapper interface {
	Run(ctx context.Context) error
}

type JavaD14Mapper interface {
	Run(ctx context.Context) error
}

type JavaD15Mapper interface {
	Run(ctx context.Context) error
}

type JavaD16Mapper interface {
	Run(ctx context.Context) error
}

type JavaD17Mapper interface {
	Run(ctx context.Context) error
}

type JavaD18Mapper interface {
	Run(ctx context.Context) error
}

type JavaD19Mapper interface {
	Run(ctx context.Context) error
}

type JavaD20Mapper interface {
	Run(ctx context.Context) error
}

type JavaD21Mapper interface {
	Run(ctx context.Context, values []string) error
}

type JavaD23Mapper interface {
	Run(ctx context.Context, values []int64) error
}

type JavaD24Mapper interface {
	Run(
		ctx context.Context,
		uuu map[string]string,
		uuuu []ConformanceBean,
	) error
}

type JavaD25Mapper interface {
	Run(ctx context.Context, beans []ConformanceBean) error
}

type JavaD27Mapper interface {
	Run(ctx context.Context) error
}

type JavaP01Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaP02Mapper interface {
	Run(ctx context.Context) error
}

type JavaP06Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaP07Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaP11Mapper interface {
	Run(ctx context.Context) error
}

type JavaP12Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaP14Mapper interface {
	Run(ctx context.Context) error
}

type JavaX01Mapper interface {
	Run(ctx context.Context) error
}

type JavaX02Mapper interface {
	Run(ctx context.Context) error
}

type JavaE01Mapper interface {
	Run(ctx context.Context, username string) error
}

type JavaE02Mapper interface {
	Run(ctx context.Context, username string) error
}

type JavaE03Mapper interface {
	Run(ctx context.Context, username *string) error
}

type JavaE04Mapper interface {
	Run(ctx context.Context, password *string) error
}

type JavaE05Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaE06Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaE07Mapper interface {
	Run(ctx context.Context, value float64) error
}

type JavaE08Mapper interface {
	Run(ctx context.Context, values []string) error
}

type JavaE09Mapper interface {
	Run(ctx context.Context, value string) error
}

type JavaE10Mapper interface {
	Run(ctx context.Context, id int64) error
}

type JavaE11Mapper interface {
	Run(ctx context.Context, name string) error
}

type JavaE12Mapper interface {
	Run(ctx context.Context, name string) error
}

type JavaE13Mapper interface {
	Run(ctx context.Context, name string) error
}

type JavaE14Mapper interface {
	Run(ctx context.Context, value *string) error
}

type ConformanceEnum string

const (
	ConformanceEmployee ConformanceEnum = "EMPLOYEE"
	ConformanceDirector ConformanceEnum = "DIRECTOR"
)

func (value ConformanceEnum) String() string {
	return string(value)
}

type JavaE15Mapper interface {
	Run(ctx context.Context, value ConformanceEnum) error
}

func EscapeLike(value string) string {
	return "%" + value + "%"
}

type JavaE16Mapper interface {
	Run(ctx context.Context, value string) error
}

type conformancePrivateData struct {
	id int
}

type JavaO01Mapper interface {
	Run(ctx context.Context, data conformancePrivateData) error
}

type ConformanceDate int

func (value ConformanceDate) Compare(other ConformanceDate) int {
	switch {
	case value < other:
		return -1
	case value > other:
		return 1
	default:
		return 0
	}
}

type JavaO02Mapper interface {
	Run(ctx context.Context, left, right ConformanceDate) error
}

func ConformanceUserName(number int) string {
	return "User" + strconv.Itoa(number)
}

type JavaE17Mapper interface {
	Run(ctx context.Context, values []int) error
}

type JavaE18Mapper interface {
	Run(ctx context.Context, values map[string]string) error
}

type JavaE19Mapper interface {
	Run(ctx context.Context, values map[int]bool) error
}

type ConformancePair struct {
	A int
	B string
}

type JavaE20Mapper interface {
	Run(ctx context.Context, values map[ConformancePair]ConformancePair) error
}

type ConformanceName struct {
	LastName   string
	FirstNames []string
}

type JavaE21Mapper interface {
	Run(ctx context.Context, names []ConformanceName) error
}

type JavaE22Mapper interface {
	Run(ctx context.Context, name string) error
}

type JavaE23Mapper interface {
	Run(ctx context.Context, schema yourbatis.Identifier, ids []int64) error
}

type JavaE24Mapper interface {
	Run(ctx context.Context, values map[yourbatis.Identifier]int) error
}

type JavaE25Mapper interface {
	Run(ctx context.Context, firstAttr, secondAttr []string) error
}

type JavaD26Mapper interface {
	Run(ctx context.Context, id *string) error
}
