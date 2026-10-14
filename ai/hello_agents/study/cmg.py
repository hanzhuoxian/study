from contextlib import contextmanager

@contextmanager
def b():
    print("Entering context")
    try: 
        yield 2
    finally:
        print("Exiting context")


with b() as x:
    print("Inside context")
    print(f"Value from context: {x}")