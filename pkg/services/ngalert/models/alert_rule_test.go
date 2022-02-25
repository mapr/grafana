package models

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/util/cmputil"
)

func TestNoDataStateFromString(t *testing.T) {
	allKnownNoDataStates := [...]NoDataState{
		Alerting,
		NoData,
		OK,
	}

	t.Run("should parse known values", func(t *testing.T) {
		for _, state := range allKnownNoDataStates {
			stateStr := string(state)
			actual, err := NoDataStateFromString(stateStr)
			require.NoErrorf(t, err, "failed to parse a known state [%s]", stateStr)
			require.Equal(t, state, actual)
		}
	})

	t.Run("should fail to parse in different case", func(t *testing.T) {
		for _, state := range allKnownNoDataStates {
			stateStr := strings.ToLower(string(state))
			actual, err := NoDataStateFromString(stateStr)
			require.Errorf(t, err, "expected error for input value [%s]", stateStr)
			require.Equal(t, NoDataState(""), actual)
		}
	})

	t.Run("should fail to parse unknown values", func(t *testing.T) {
		input := util.GenerateShortUID()
		actual, err := NoDataStateFromString(input)
		require.Errorf(t, err, "expected error for input value [%s]", input)
		require.Equal(t, NoDataState(""), actual)
	})
}

func TestErrStateFromString(t *testing.T) {
	allKnownErrStates := [...]ExecutionErrorState{
		AlertingErrState,
		ErrorErrState,
		OkErrState,
	}

	t.Run("should parse known values", func(t *testing.T) {
		for _, state := range allKnownErrStates {
			stateStr := string(state)
			actual, err := ErrStateFromString(stateStr)
			require.NoErrorf(t, err, "failed to parse a known state [%s]", stateStr)
			require.Equal(t, state, actual)
		}
	})

	t.Run("should fail to parse in different case", func(t *testing.T) {
		for _, state := range allKnownErrStates {
			stateStr := strings.ToLower(string(state))
			actual, err := ErrStateFromString(stateStr)
			require.Errorf(t, err, "expected error for input value [%s]", stateStr)
			require.Equal(t, ExecutionErrorState(""), actual)
		}
	})

	t.Run("should fail to parse unknown values", func(t *testing.T) {
		input := util.GenerateShortUID()
		actual, err := ErrStateFromString(input)
		require.Errorf(t, err, "expected error for input value [%s]", input)
		require.Equal(t, ExecutionErrorState(""), actual)
	})
}

func TestPatchPartialAlertRule(t *testing.T) {
	t.Run("patches", func(t *testing.T) {
		testCases := []struct {
			name    string
			mutator func(r *AlertRule)
		}{
			{
				name: "title is empty",
				mutator: func(r *AlertRule) {
					r.Title = ""
				},
			},
			{
				name: "condition and data are empty",
				mutator: func(r *AlertRule) {
					r.Condition = ""
					r.Data = nil
				},
			},
			{
				name: "ExecErrState is empty",
				mutator: func(r *AlertRule) {
					r.ExecErrState = ""
				},
			},
			{
				name: "NoDataState is empty",
				mutator: func(r *AlertRule) {
					r.NoDataState = ""
				},
			},
			{
				name: "For is 0",
				mutator: func(r *AlertRule) {
					r.For = 0
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				var existing *AlertRule
				for {
					existing = AlertRuleGen(func(rule *AlertRule) {
						rule.For = time.Duration(rand.Int63n(1000) + 1)
					})()
					cloned := *existing
					testCase.mutator(&cloned)
					if !cmp.Equal(*existing, cloned, cmp.FilterPath(func(path cmp.Path) bool {
						return path.String() == "Data.modelProps"
					}, cmp.Ignore())) {
						break
					}
				}
				patch := *existing
				testCase.mutator(&patch)

				require.NotEqual(t, *existing, patch)
				PatchPartialAlertRule(existing, &patch)
				require.Equal(t, *existing, patch)
			})
		}
	})

	t.Run("does not patch", func(t *testing.T) {
		testCases := []struct {
			name    string
			mutator func(r *AlertRule)
		}{
			{
				name: "ID",
				mutator: func(r *AlertRule) {
					r.ID = 0
				},
			},
			{
				name: "OrgID",
				mutator: func(r *AlertRule) {
					r.OrgID = 0
				},
			},
			{
				name: "Updated",
				mutator: func(r *AlertRule) {
					r.Updated = time.Time{}
				},
			},
			{
				name: "Version",
				mutator: func(r *AlertRule) {
					r.Version = 0
				},
			},
			{
				name: "UID",
				mutator: func(r *AlertRule) {
					r.UID = ""
				},
			},
			{
				name: "DashboardUID",
				mutator: func(r *AlertRule) {
					r.DashboardUID = nil
				},
			},
			{
				name: "PanelID",
				mutator: func(r *AlertRule) {
					r.PanelID = nil
				},
			},
			{
				name: "Annotations",
				mutator: func(r *AlertRule) {
					r.Annotations = nil
				},
			},
			{
				name: "Labels",
				mutator: func(r *AlertRule) {
					r.Labels = nil
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				var existing *AlertRule
				for {
					existing = AlertRuleGen()()
					cloned := *existing
					// make sure the generated rule does not match the mutated one
					testCase.mutator(&cloned)
					if !cmp.Equal(*existing, cloned, cmp.FilterPath(func(path cmp.Path) bool {
						return path.String() == "Data.modelProps"
					}, cmp.Ignore())) {
						break
					}
				}
				patch := *existing
				testCase.mutator(&patch)
				PatchPartialAlertRule(existing, &patch)
				require.NotEqual(t, *existing, patch)
			})
		}
	})
}

func TestDiff(t *testing.T) {
	simpleDiff := func(field string) func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
		return func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
			result := false
			if assert.Len(t, d.Diffs, 1) {
				diff := d.Diffs[0]
				result = assert.Equal(t, field, diff.Path)

				origVal := reflect.Indirect(reflect.ValueOf(orig))
				changeVal := reflect.Indirect(reflect.ValueOf(change))

				origField := origVal.FieldByName(field)
				changeField := changeVal.FieldByName(field)

				if origField.Kind() == reflect.Ptr {
					checkPtr := func(expected, actual reflect.Value) bool {
						if expected.IsNil() {
							return assert.Nil(t, actual.Interface())
						} else {
							if !assert.NotNil(t, actual.Interface()) {
								return false
							}
							if actual.Kind() == reflect.Ptr {
								actual = actual.Elem()
							}
							return assert.Equal(t, expected.Elem().Interface(), actual.Interface())
						}
					}
					result = result && checkPtr(origField, diff.Left)
					result = result && checkPtr(changeField, diff.Right)
				} else {
					result = result && assert.Equal(t, origField.Interface(), diff.Left.Interface())
					result = result && assert.Equal(t, changeField.Interface(), diff.Right.Interface())
				}
			}

			if !result {
				t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
			}
		}
	}

	testCases := []struct {
		name     string
		copyRule func(r *AlertRule) *AlertRule
		assert   func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport)
	}{
		{
			name: "should not be diff if same fields",
			copyRule: func(r *AlertRule) *AlertRule {
				return CopyRule(r)
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				if d != nil {
					t.Fatalf("expected nil but got: %s", d)
				}
			},
		},
		{
			name: "should detect changes in ID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.ID == rule.ID {
					rule.ID = rand.Int63()
				}
				return rule
			},
			assert: simpleDiff("ID"),
		},
		{
			name: "should detect changes in OrgID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.OrgID == rule.OrgID {
					rule.OrgID = rand.Int63()
				}
				return rule
			},
			assert: simpleDiff("OrgID"),
		},
		{
			name: "should detect changes in Title",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.Title == rule.Title {
					r.Title = util.GenerateShortUID()
				}
				return rule
			},
			assert: simpleDiff("Title"),
		},
		{
			name: "should detect changes in Condition",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.Condition == rule.Condition {
					r.Condition = util.GenerateShortUID()
				}
				return rule
			},
			assert: simpleDiff("Condition"),
		},
		{
			name: "should detect changes in Updated",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.Updated == rule.Updated {
					r.Updated = r.Updated.Add(time.Duration(rand.Int()))
				}
				return rule
			},
			assert: simpleDiff("Updated"),
		},
		{
			name: "should detect changes in IntervalSeconds",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.IntervalSeconds == rule.IntervalSeconds {
					r.IntervalSeconds = rand.Int63()
				}
				return rule
			},
			assert: simpleDiff("IntervalSeconds"),
		},
		{
			name: "should detect changes in Version",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.Version == rule.Version {
					r.Version = rand.Int63()
				}
				return rule
			},
			assert: simpleDiff("Version"),
		},
		{
			name: "should detect changes in UID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.UID == rule.UID {
					r.UID = util.GenerateShortUID()
				}
				return rule
			},
			assert: simpleDiff("UID"),
		},
		{
			name: "should detect changes in NamespaceUID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.NamespaceUID == rule.NamespaceUID {
					r.NamespaceUID = util.GenerateShortUID()
				}
				return rule
			},
			assert: simpleDiff("NamespaceUID"),
		},
		{
			name: "should detect changes in DashboardUID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				if r.DashboardUID == nil {
					d := util.GenerateShortUID()
					rule.DashboardUID = &d
				} else if rand.Int63()%2 == 0 {
					rule.DashboardUID = nil
				} else {
					d := util.GenerateShortUID()
					for d == *r.DashboardUID {
						d = util.GenerateShortUID()
					}
					rule.DashboardUID = &d
				}
				return rule
			},
			assert: simpleDiff("DashboardUID"),
		},
		{
			name: "should detect changes in PanelID",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				if r.PanelID == nil {
					d := rand.Int63()
					rule.PanelID = &d
				} else if rand.Int63()%2 == 0 {
					r.PanelID = nil
				} else {
					d := rand.Int63()
					for d == *r.PanelID {
						d = rand.Int63()
					}
					rule.PanelID = &d
				}
				return rule
			},
			assert: simpleDiff("PanelID"),
		},
		{
			name: "should detect changes in RuleGroup",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.RuleGroup == rule.RuleGroup {
					r.RuleGroup = util.GenerateShortUID()
				}
				return rule
			},
			assert: simpleDiff("RuleGroup"),
		},
		{
			name: "should detect changes in NoDataState",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.NoDataState == rule.NoDataState {
					rule.NoDataState = NoDataState(util.GenerateShortUID())
				}
				return rule
			},
			assert: simpleDiff("NoDataState"),
		},
		{
			name: "should detect changes in ExecErrState",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.ExecErrState == rule.ExecErrState {
					rule.ExecErrState = ExecutionErrorState(util.GenerateShortUID())
				}
				return rule
			},
			assert: simpleDiff("ExecErrState"),
		},
		{
			name: "should detect changes in For",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				for r.For == rule.For {
					rule.For = time.Duration(rand.Int63())
				}
				return rule
			},
			assert: simpleDiff("For"),
		},
		{
			name: "should detect changes in Annotation values",
			copyRule: func(r *AlertRule) *AlertRule {
				if len(r.Annotations) == 0 {
					r.Annotations = make(map[string]string, 8)
					for i := 0; i < rand.Intn(5)+3; i++ {
						r.Annotations[util.GenerateShortUID()] = util.GenerateShortUID()
					}
				}
				rule := CopyRule(r)
				for key, value := range rule.Annotations {
					v := util.GenerateShortUID()
					for v == value {
						v = util.GenerateShortUID()
					}
					rule.Annotations[key] = v
				}
				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				if assert.Len(t, d.Diffs, len(orig.Annotations)) {
					result = true
				outerloop:
					for key, expected := range orig.Annotations {
						expectedKey := fmt.Sprintf("Annotations[%s]", key)
						expectedNew := change.Annotations[key]
						for _, diff := range d.Diffs {
							if diff.Path == expectedKey {
								result = result && assert.Equal(t, expected, diff.Left.String())
								result = result && assert.Equal(t, expectedNew, diff.Right.String())
								continue outerloop
							}
						}
						result = result && assert.Fail(t, "path %s does not exist in diff", expectedKey)
					}
				}
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect new keys in Annotation",
			copyRule: func(r *AlertRule) *AlertRule {
				if len(r.Annotations) == 0 {
					r.Annotations = make(map[string]string, 8)
					for i := 0; i < rand.Intn(5)+3; i++ {
						r.Annotations[util.GenerateShortUID()] = util.GenerateShortUID()
					}
				}
				rule := CopyRule(r)
				for i := 0; i < rand.Intn(5)+1; i++ {
					key := util.GenerateShortUID()
					for _, ok := rule.Annotations[key]; ok; {
						key = util.GenerateShortUID()
					}
					rule.Annotations[key] = util.GenerateShortUID()
				}
				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)

				if assert.Len(t, d.Diffs, len(change.Annotations)-len(orig.Annotations)) {
					result = true
				outerloop:
					for key, expectedNew := range change.Annotations {
						if _, ok := orig.Annotations[key]; ok {
							continue
						}
						expectedKey := fmt.Sprintf("Annotations[%s]", key)
						for _, diff := range d.Diffs {
							if diff.Path == expectedKey {
								result = result && assert.False(t, diff.Left.IsValid())
								result = result && assert.Equal(t, expectedNew, diff.Right.String())
								continue outerloop
							}
						}
						result = result && assert.Fail(t, "path %s does not exist in diff", expectedKey)
					}
				}
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect changes in Labels values",
			copyRule: func(r *AlertRule) *AlertRule {
				if len(r.Labels) == 0 {
					r.Labels = make(map[string]string, 8)
					for i := 0; i < rand.Intn(5)+3; i++ {
						r.Labels[util.GenerateShortUID()] = util.GenerateShortUID()
					}
				}
				rule := CopyRule(r)
				for key, value := range rule.Labels {
					v := util.GenerateShortUID()
					for v == value {
						v = util.GenerateShortUID()
					}
					rule.Labels[key] = v
				}
				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				if assert.Len(t, d.Diffs, len(orig.Labels)) {
					result = true
				outerloop:
					for key, expected := range orig.Labels {
						expectedKey := fmt.Sprintf("Labels[%s]", key)
						expectedNew := change.Labels[key]
						for _, diff := range d.Diffs {
							if diff.Path == expectedKey {
								result = result && assert.Equal(t, expected, diff.Left.String())
								result = result && assert.Equal(t, expectedNew, diff.Right.String())
								continue outerloop
							}
						}
						result = result && assert.Fail(t, "path %s does not exist in diff", expectedKey)
					}
				}
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect new keys in Labels",
			copyRule: func(r *AlertRule) *AlertRule {
				if len(r.Labels) == 0 {
					r.Labels = make(map[string]string, 8)
					for i := 0; i < rand.Intn(5)+3; i++ {
						r.Labels[util.GenerateShortUID()] = util.GenerateShortUID()
					}
				}
				rule := CopyRule(r)
				for i := 0; i < rand.Intn(5)+1; i++ {
					key := util.GenerateShortUID()
					for _, ok := rule.Labels[key]; ok; {
						key = util.GenerateShortUID()
					}
					rule.Labels[key] = util.GenerateShortUID()
				}
				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)

				if assert.Len(t, d.Diffs, len(change.Labels)-len(orig.Labels)) {
					result = true
				outerloop:
					for key, expectedNew := range change.Labels {
						if _, ok := orig.Labels[key]; ok {
							continue
						}
						expectedKey := fmt.Sprintf("Labels[%s]", key)
						for _, diff := range d.Diffs {
							if diff.Path == expectedKey {
								result = result && assert.False(t, diff.Left.IsValid())
								result = result && assert.Equal(t, expectedNew, diff.Right.String())
								continue outerloop
							}
						}
						result = result && assert.Fail(t, "path %s does not exist in diff", expectedKey)
					}
				}
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect changes in Data",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				rule.Data = append(rule.Data, GenerateAlertQuery())
				rule.Data = append([]AlertQuery{
					GenerateAlertQuery(),
				}, rule.Data...)

				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect changes in Data",
			copyRule: func(r *AlertRule) *AlertRule {
				rule := CopyRule(r)
				r.Data = append(r.Data, GenerateAlertQuery())
				rule.Data = append([]AlertQuery{
					GenerateAlertQuery(),
				}, rule.Data...)

				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
		{
			name: "should detect changes in Data",
			copyRule: func(r *AlertRule) *AlertRule {
				r.Data = append(r.Data, GenerateAlertQuery(), GenerateAlertQuery())
				rule := CopyRule(r)
				rule.Data[0].QueryType = util.GenerateShortUID()
				// rule.Data[0].RefID = util.GenerateShortUID()
				rule.Data[0].DatasourceUID = util.GenerateShortUID()
				rule.Data[0].Model = json.RawMessage(`{ "test": "test" }`)
				return rule
			},
			assert: func(t *testing.T, orig, change *AlertRule, d *cmputil.DiffReport) {
				result := false
				if !result {
					t.Logf("rule1: %#v, rule2: %#v\ndiff: %s", orig, change, d)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rule1 := AlertRuleGen()()
			rule2 := testCase.copyRule(rule1)
			result := rule1.Diff(rule2)
			testCase.assert(t, rule1, rule2, result)
		})
	}
}
