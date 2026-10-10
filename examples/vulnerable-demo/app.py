"""Demo app — imports cryptography so SCA scanners attribute the pin."""

try:
    import cryptography  # noqa: F401
except ImportError:
    pass


def main() -> None:
    print("repository-detective vulnerable-demo")


if __name__ == "__main__":
    main()
