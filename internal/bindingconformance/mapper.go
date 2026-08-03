package bindingconformance

import (
	"context"

	yourbatis "github.com/superduck-ai/yourbatis"
)

type BindingBlog struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
}

type BindingPost struct {
	ID      int64  `db:"id"`
	BlogID  int64  `db:"blog_id"`
	Subject string `db:"subject"`
	Body    string `db:"body"`
}

type BindingAuthor struct {
	ID       int64  `db:"id"`
	Username string `db:"username"`
	Password string `db:"password"`
	Email    string `db:"email"`
	Bio      string `db:"bio"`
}

// Each mapper below corresponds to one independently migratable method from
// MyBatis BindingTest. Keeping one interface and one XML file per source method
// prevents broad smoke tests from hiding an unsupported or untested behavior.

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB02Mapper -sql ./fixtures/b02.xml
type BindingB02Mapper interface {
	Run(ctx context.Context, ids []int64) ([]BindingPost, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB03Mapper -sql ./fixtures/b03.xml
type BindingB03Mapper interface {
	Run(ctx context.Context, ids [3]int64) ([]BindingPost, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB04Mapper -sql ./fixtures/b04.xml
type BindingB04Mapper interface {
	Run(
		ctx context.Context,
		one int64,
		two int64,
		three int64,
		offset int64,
		limit int64,
	) ([]BindingPost, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB07Mapper -sql ./fixtures/b07.xml
type BindingB07Mapper interface {
	Run(ctx context.Context, author BindingAuthor) (int64, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB09Mapper -sql ./fixtures/b09.xml
type BindingB09Mapper interface {
	Run(ctx context.Context) (int64, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB10Mapper -sql ./fixtures/b10.xml
type BindingB10Mapper interface {
	Run(ctx context.Context) ([]BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB12Mapper -sql ./fixtures/b12.xml
type BindingB12Mapper interface {
	Run(ctx context.Context, yield func(BindingBlog) error) error
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB13Mapper -sql ./fixtures/b13.xml
type BindingB13Mapper interface {
	Run(ctx context.Context, yield func(BindingBlog) error) error
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB16Mapper -sql ./fixtures/b16.xml
type BindingB16Mapper interface {
	Run(ctx context.Context) ([]BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB19Mapper -sql ./fixtures/b19.xml
type BindingB19Mapper interface {
	Run(
		ctx context.Context,
		query string,
		offset int64,
		limit int64,
	) ([]BindingPost, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB20Mapper -sql ./fixtures/b20.xml
type BindingB20Mapper interface {
	Run(
		ctx context.Context,
		subjectQuery string,
		bodyQuery string,
		offset int64,
		limit int64,
	) ([]BindingPost, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB21Mapper -sql ./fixtures/b21.xml
type BindingB21Mapper interface {
	Run(ctx context.Context, id int64) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB28Mapper -sql ./fixtures/b28.xml
type BindingB28Mapper interface {
	Run(ctx context.Context, id int64) (BindingAuthor, error)
}

// B36 is a generation-failure fixture and deliberately has no go:generate
// directive. The test invokes generator.Generate and verifies the diagnostic.
type BindingB36Mapper interface {
	Run(ctx context.Context, id int64) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB37Mapper -sql ./fixtures/b37.xml
type BindingB37Mapper interface {
	Run(ctx context.Context, id *int64) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB38Mapper -sql ./fixtures/b38.xml
type BindingB38Mapper interface {
	Run(ctx context.Context, params map[string]string) (BindingBlog, bool, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB39Mapper -sql ./fixtures/b39.xml
type BindingB39Mapper interface {
	Run(ctx context.Context, id int64, title string) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB40Mapper -sql ./fixtures/b40.xml
type BindingB40Mapper interface {
	Run(ctx context.Context, id int64, title string) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB41Mapper -sql ./fixtures/b41.xml
type BindingB41Mapper interface {
	Run(
		ctx context.Context,
		column yourbatis.Identifier,
		id int64,
		title string,
	) (BindingBlog, error)
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB45Mapper -sql ./fixtures/b45.xml
type BindingB45Mapper interface {
	Run(
		ctx context.Context,
		offset int64,
		limit int64,
		yield func(BindingBlog) error,
	) error
}

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen -mapper BindingB47Mapper -sql ./fixtures/b47.xml
type BindingB47Mapper interface {
	Run(
		ctx context.Context,
		offset int64,
		limit int64,
		yield func(BindingBlog) error,
	) error
}
