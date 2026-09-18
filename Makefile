TASKS := $(sort $(notdir $(wildcard tasks/task_*)))

# Same flags as the official check (cmd/grade).
TEST_FLAGS := -race -count=4 -timeout=60s

# The grader snapshots the whole candidate tree, so its output must live
# outside the repository.
GRADE_OUT ?= $(or $(TMPDIR),/tmp)/autumn-grade

.PHONY: help test check grade $(TASKS) $(addprefix run-,$(TASKS))

help:
	@echo "make test          quick run of all tests"
	@echo "make check         all tasks, as in the assignment: go test $(TEST_FLAGS)"
	@echo "make task_XX       one task, as in the assignment (e.g. make task_05)"
	@echo "make run-task_XX   run the task demo (go run ./tasks/task_XX)"
	@echo "make grade         official grader locally"

test:
	go test ./...

check:
	go test $(TEST_FLAGS) ./tasks/...

$(TASKS):
	go test $(TEST_FLAGS) ./tasks/$@

$(addprefix run-,$(TASKS)):
	go run ./tasks/$(@:run-%=%)

grade:
	go run ./cmd/grade --baseline . --candidate . --out $(GRADE_OUT) --local
	@echo "results: $(GRADE_OUT)/package-results.json"
