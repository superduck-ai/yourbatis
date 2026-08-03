package mybatiscoreconformance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/superduck-ai/yourbatis/internal/generator"
)

func TestMigratedGenerationFailures(t *testing.T) {
	testCases := []struct {
		name       string
		mapper     string
		fixture    string
		want       string
		wantSource bool
	}{
		{
			name:       "WrongNamespacesTest_shouldFailForWrongNamespace",
			mapper:     "WrongNamespaceMapper",
			fixture:    "wrong_namespace.xml",
			want:       `mapper namespace "SomeOtherMapper" does not match interface "WrongNamespaceMapper"`,
			wantSource: true,
		},
		{
			name:       "WrongNamespacesTest_shouldFailForMissingNamespace",
			mapper:     "MissingNamespaceMapper",
			fixture:    "missing_namespace.xml",
			want:       "mapper requires a non-empty namespace",
			wantSource: true,
		},
		{
			name:       "XmlMapperBuilderTest_errorResultMapLocation",
			mapper:     "MissingResultMapMapper",
			fixture:    "missing_result_map.xml",
			want:       `resultMap "MissingResultMapMapper.doesNotExist" not found`,
			wantSource: true,
		},
		{
			name:    "NoResultTypeMapTest_shouldFailIfNoMatchingMethod",
			mapper:  "ExtraStatementMapper",
			fixture: "extra_statement.xml",
			want:    "SQL statements have no matching mapper methods: NoMatchingMethod",
		},
		{
			name:    "MapperExtendTest_shouldThrowExceptionIfNoMatchingStatementFound",
			mapper:  "MissingStatementMapper",
			fixture: "missing_statement.xml",
			want:    "mapper method MissingStatementMapper.Unmapped has no matching SQL statement",
		},
		{
			name:       "RefidResolutionTest_includes",
			mapper:     "MissingIncludeMapper",
			fixture:    "missing_include.xml",
			want:       `SQL fragment "other.Mapper.columns" not found`,
			wantSource: true,
		},
		{
			name:       "NamedResultMapTest_namedResultMapWithNoMappingsShouldFail",
			mapper:     "EmptyResultMapMapper",
			fixture:    "empty_result_map.xml",
			want:       "resultMap EmptyResultMapMapper.empty must declare at least one field",
			wantSource: true,
		},
		{
			name:       "IdConflictTest_shouldFailOnDuplicatedId",
			mapper:     "DuplicateResultMapMapper",
			fixture:    "duplicate_result_map.xml",
			want:       `duplicate resultMap "DuplicateResultMapMapper.duplicate"`,
			wantSource: true,
		},
		{
			name:       "SQLFragmentIdCollision_qualifiedThenBare",
			mapper:     "DuplicateFragmentQualifiedThenBareMapper",
			fixture:    "duplicate_fragment_qualified_then_bare.xml",
			want:       `duplicate sql fragment "DuplicateFragmentQualifiedThenBareMapper.columns"`,
			wantSource: true,
		},
		{
			name:       "SQLFragmentIdCollision_bareThenQualified",
			mapper:     "DuplicateFragmentBareThenQualifiedMapper",
			fixture:    "duplicate_fragment_bare_then_qualified.xml",
			want:       `duplicate sql fragment "DuplicateFragmentBareThenQualifiedMapper.columns"`,
			wantSource: true,
		},
		{
			name:       "ResultMapIdCollision_qualifiedThenBare",
			mapper:     "DuplicateResultMapQualifiedThenBareMapper",
			fixture:    "duplicate_result_map_qualified_then_bare.xml",
			want:       `duplicate resultMap "DuplicateResultMapQualifiedThenBareMapper.result"`,
			wantSource: true,
		},
		{
			name:       "ResultMapIdCollision_bareThenQualified",
			mapper:     "DuplicateResultMapBareThenQualifiedMapper",
			fixture:    "duplicate_result_map_bare_then_qualified.xml",
			want:       `duplicate resultMap "DuplicateResultMapBareThenQualifiedMapper.result"`,
			wantSource: true,
		},
		{
			name:       "ResultMapDefinition_rejectsForeignNamespace",
			mapper:     "ForeignResultMapDefinitionMapper",
			fixture:    "foreign_result_map_definition.xml",
			want:       "dots are not allowed in resultMap ids",
			wantSource: true,
		},
		{
			name:       "ResultMapReference_doesNotFallBackToBareName",
			mapper:     "ForeignResultMapReferenceMapper",
			fixture:    "foreign_result_map_reference.xml",
			want:       `resultMap "OtherMapper.result" not found`,
			wantSource: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := filepath.Join("negative", testCase.fixture)
			output := filepath.Join(t.TempDir(), "must_not_exist.go")
			err := generator.Generate(generator.Config{
				PackageDir: ".",
				Mapper:     testCase.mapper,
				SQLFile:    fixture,
				Output:     output,
			})
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Generate() error = %v, want substring %q", err, testCase.want)
			}
			if testCase.wantSource && !strings.Contains(err.Error(), fixture) {
				t.Fatalf("Generate() error = %v, want source %q", err, fixture)
			}
			if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("failed generation left output %q: %v", output, statErr)
			}
		})
	}
}
