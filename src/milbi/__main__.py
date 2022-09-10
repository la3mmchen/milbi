
#!/usr/bin/env python3

import fire
import sys
from .class_milbi import Milbi


def main():
    try:
        fire.Fire(Milbi, name="milbi")
    except Exception as e:
        print(e)
        sys.exit(1)
    sys.exit(0)


if __name__ == '__main__':
    """
    milbi
    """
    main()
