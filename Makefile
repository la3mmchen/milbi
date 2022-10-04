.DEFAULT_GOAL := help
PYTEST_FLAGS := -x -n 1 -v --pyargs .
COVERAGE_FLAGS := --junitxml=pytest-report.xml
PYTEST_INSTALLS := pytest==7.1.3 pytest-cov==3.0.0 pytest-forked==1.4.0 pytest-xdist==2.5.0


.PHONY: lint
lint: ## lint with flake8
lint:
	python3 -m pip install flake8
	flake8 --config .flake8 milbi.py
	flake8 --config .flake8 src/*/*.py

.PHONY: tests
tests: ## execute all test cases
tests: testprepare unittests clitests

.PHONE: testprepare
testprepare: 
testprepare:
	@python3 -m pip install ${PYTEST_INSTALLS}

.PHONY: unittests
unittests: ## execute unittests
unittests:
	@echo "running test limited to: ${PYTEST_FILTER}"
	coverage run --source=src -m pytest ${PYTEST_FLAGS} -k test_unit ${COVERAGE_FLAGS}
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
