TASKS := $(sort $(notdir $(wildcard tasks/task_*)))

# Same flags as the official check (cmd/grade).
TEST_FLAGS := -race -count=4 -timeout=60s

# The grader snapshots the whole candidate tree, so its output must live
# outside the repository.
GRADE_OUT ?= $(or $(TMPDIR),/tmp)/autumn-grade

.PHONY: help test check grade $(TASKS) $(addprefix run-,$(TASKS))

help:
	@echo "make test          go test ./... without -race and -count"
	@echo "make check         all tasks with the grader flags: go test $(TEST_FLAGS)"
	@echo "make task_XX       one task with the grader flags, e.g. make task_05"
	@echo "make run-task_XX   go run ./tasks/task_XX"
	@echo "make grade         run the grader locally, without Docker"

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
