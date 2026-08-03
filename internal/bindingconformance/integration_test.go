package bindingconformance

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	yourbatis "github.com/superduck-ai/yourbatis"
)

func runMigratedBindingTests(t *testing.T, executor yourbatis.Executor) {
	t.Helper()
	ctx := context.Background()

	t.Run("B02_shouldFindPostsInList", func(t *testing.T) {
		posts, err := NewBindingB02Mapper(executor).Run(ctx, []int64{1, 3, 5})
		if err != nil {
			t.Fatal(err)
		}
		assertBindingPostIDs(t, posts, 1, 3, 5)
	})

	t.Run("B03_shouldFindPostsInArray", func(t *testing.T) {
		posts, err := NewBindingB03Mapper(executor).Run(ctx, [3]int64{1, 3, 5})
		if err != nil {
			t.Fatal(err)
		}
		assertBindingPostIDs(t, posts, 1, 3, 5)
	})

	t.Run("B04_shouldFindThreeSpecificPosts", func(t *testing.T) {
		posts, err := NewBindingB04Mapper(executor).Run(ctx, 1, 3, 5, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		assertBindingPostIDs(t, posts, 3)
	})

	t.Run("B07_verifyErrorMessageFromInsertAfterSelectKey", func(t *testing.T) {
		_, err := NewBindingB07Mapper(executor).Run(ctx, BindingAuthor{
			ID:       201,
			Username: "invalid insert",
			Password: "secret",
			Email:    "invalid@example.com",
			Bio:      "invalid",
		})
		var statementErr *yourbatis.StatementError
		if !errors.As(err, &statementErr) {
			t.Fatalf("error = %T %v, want *yourbatis.StatementError", err, err)
		}
		if statementErr.Statement.ID != "BindingB07Mapper.Run" ||
			!strings.Contains(statementErr.Statement.Source, "fixtures/b07.xml") {
			t.Fatalf("statement error = %#v", statementErr)
		}
	})

	t.Run("B09_shouldSelectRandom", func(t *testing.T) {
		value, err := NewBindingB09Mapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if value != 1 {
			t.Fatalf("scalar value = %d, want 1", value)
		}
	})

	t.Run("B10_shouldExecuteBoundSelectListOfBlogsStatement", func(t *testing.T) {
		blogs, err := NewBindingB10Mapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assertBindingBlogs(t, blogs)
	})

	t.Run("B12_resultHandlerBetweenSessions", func(t *testing.T) {
		first := collectBindingBlogs(t, ctx, NewBindingB12Mapper(executor).Run)
		second := collectBindingBlogs(t, ctx, NewBindingB12Mapper(executor).Run)
		assertBindingBlogs(t, first)
		assertBindingBlogs(t, second)
	})

	t.Run("B13_resultHandlerInSameSession", func(t *testing.T) {
		mapper := NewBindingB13Mapper(executor)
		first := collectBindingBlogs(t, ctx, mapper.Run)
		second := collectBindingBlogs(t, ctx, mapper.Run)
		assertBindingBlogs(t, first)
		assertBindingBlogs(t, second)
	})

	t.Run("B16_shouldSelectListOfBlogsUsingXMLConfig", func(t *testing.T) {
		blogs, err := NewBindingB16Mapper(executor).Run(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assertBindingBlogs(t, blogs)
	})

	t.Run("B19_shouldSelectListOfPostsLike", func(t *testing.T) {
		posts, err := NewBindingB19Mapper(executor).Run(ctx, "%a%", 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		assertBindingPostIDs(t, posts, 2)
	})

	t.Run("B20_shouldSelectListOfPostsLikeTwoParameters", func(t *testing.T) {
		posts, err := NewBindingB20Mapper(executor).Run(ctx, "%a%", "%a%", 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		assertBindingPostIDs(t, posts, 2)
	})

	t.Run("B21_shouldExecuteBoundSelectOneBlogStatement", func(t *testing.T) {
		blog, err := NewBindingB21Mapper(executor).Run(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if blog.ID != 1 || blog.Title != "Jim Business" {
			t.Fatalf("blog = %#v", blog)
		}
	})

	t.Run("B28_shouldSelectOneAuthor", func(t *testing.T) {
		author, err := NewBindingB28Mapper(executor).Run(ctx, 101)
		if err != nil {
			t.Fatal(err)
		}
		want := BindingAuthor{
			ID:       101,
			Username: "jim",
			Password: "********",
			Email:    "jim@ibatis.apache.org",
			Bio:      "",
		}
		if author != want {
			t.Fatalf("author = %#v, want %#v", author, want)
		}
	})

	t.Run("B37_shouldFailWhenSelectingOneBlogWithNullParam", func(t *testing.T) {
		_, err := NewBindingB37Mapper(executor).Run(ctx, nil)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("error = %v, want sql.ErrNoRows", err)
		}
		var statementErr *yourbatis.StatementError
		if !errors.As(err, &statementErr) || statementErr.Statement.ID != "BindingB37Mapper.Run" {
			t.Fatalf("error = %#v, want BindingB37Mapper.Run StatementError", err)
		}
	})

	t.Run("B38_missingNestedMapParamDoesNotFailBinding", func(t *testing.T) {
		blog, found, err := NewBindingB38Mapper(executor).Run(ctx, map[string]string{})
		if err != nil || found || blog != (BindingBlog{}) {
			t.Fatalf("result = (%#v, %t, %v), want zero, false, nil", blog, found, err)
		}
	})

	t.Run("B39_shouldSelectBlogWithDefault30ParamNames", func(t *testing.T) {
		blog, err := NewBindingB39Mapper(executor).Run(ctx, 1, "Jim Business")
		if err != nil {
			t.Fatal(err)
		}
		if blog.ID != 1 || blog.Title != "Jim Business" {
			t.Fatalf("blog = %#v", blog)
		}
	})

	t.Run("B40_shouldSelectBlogWithDefault31ParamNames", func(t *testing.T) {
		blog, err := NewBindingB40Mapper(executor).Run(ctx, 1, "Jim Business")
		if err != nil {
			t.Fatal(err)
		}
		if blog.ID != 1 || blog.Title != "Jim Business" {
			t.Fatalf("blog = %#v", blog)
		}
	})

	t.Run("B41_shouldSelectBlogWithAParamNamedValue", func(t *testing.T) {
		blog, err := NewBindingB41Mapper(executor).Run(
			ctx,
			yourbatis.MustIdentifier("id"),
			1,
			"Jim Business",
		)
		if err != nil {
			t.Fatal(err)
		}
		if blog.ID != 1 || blog.Title != "Jim Business" {
			t.Fatalf("blog = %#v", blog)
		}
	})

	t.Run("B45_executeWithResultHandlerAndRowBounds", func(t *testing.T) {
		blogs := collectBindingBlogs(t, ctx, func(ctx context.Context, yield func(BindingBlog) error) error {
			return NewBindingB45Mapper(executor).Run(ctx, 1, 1, yield)
		})
		assertSingleSecondBlog(t, blogs)
	})

	t.Run("B47_executeWithCursorAndRowBounds", func(t *testing.T) {
		blogs := collectBindingBlogs(t, ctx, func(ctx context.Context, yield func(BindingBlog) error) error {
			return NewBindingB47Mapper(executor).Run(ctx, 1, 1, yield)
		})
		assertSingleSecondBlog(t, blogs)
	})
}

func collectBindingBlogs(
	t *testing.T,
	ctx context.Context,
	run func(context.Context, func(BindingBlog) error) error,
) []BindingBlog {
	t.Helper()
	var blogs []BindingBlog
	err := run(ctx, func(blog BindingBlog) error {
		blogs = append(blogs, blog)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return blogs
}

func assertBindingPostIDs(t *testing.T, posts []BindingPost, ids ...int64) {
	t.Helper()
	if len(posts) != len(ids) {
		t.Fatalf("posts = %#v, want IDs %v", posts, ids)
	}
	for index, id := range ids {
		if posts[index].ID != id {
			t.Fatalf("posts = %#v, want IDs %v", posts, ids)
		}
	}
}

func assertBindingBlogs(t *testing.T, blogs []BindingBlog) {
	t.Helper()
	if len(blogs) != 2 ||
		blogs[0] != (BindingBlog{ID: 1, Title: "Jim Business"}) ||
		blogs[1] != (BindingBlog{ID: 2, Title: "Mary Business"}) {
		t.Fatalf("blogs = %#v", blogs)
	}
}

func assertSingleSecondBlog(t *testing.T, blogs []BindingBlog) {
	t.Helper()
	if len(blogs) != 1 || blogs[0] != (BindingBlog{ID: 2, Title: "Mary Business"}) {
		t.Fatalf("blogs = %#v", blogs)
	}
}
