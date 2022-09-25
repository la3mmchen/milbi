from src.milbi.class_milbi import Milbi

def _prepare():
    return Milbi(config="tests/files/clitest-config.yaml", debug=False)


def test_unit_load_config():
    """
    execute y
    """
    obj = _prepare()

    assert isinstance(obj._load_config('tests/files/clitest-config.yaml'), dict)