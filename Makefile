.DEFAULT_GOAL := help
PYTEST_FLAGS := -x -n 1 -v --pyargs .
COVERAGE_FLAGS := --junitxml=pytest-report.xml


.PHONY: lint
lint: ## lint with flake8
lint:
	flake8 milbi.py
	flake8 src/*/*.py

.PHONY: tests
tests: ## execute all test cases
tests: unittests clitests

.PHONY: unittests
unittests: ## execute unittests
unittests:
	@echo "running test limited to: ${PYTEST_FILTER}"
	@coverage run --source=src -m pytest ${PYTEST_FLAGS} -k test_unit ${COVERAGE_FLAGS}
	@coverage report -m

.PHONY: clitests
clitests: ## execute clitests
clitests:
	@echo "running test limited to: ${PYTEST_FILTER}"
	@coverage run --source=src -m pytest ${PYTEST_FLAGS} -k test_cli ${COVERAGE_FLAGS}
	@coverage report -m

.PHONY: help
help: ## Show this help.
help:
	@echo "wrapper"
	@echo " "
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
	@echo " "
