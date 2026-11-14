import micropython
import time


@micropython.native
def fast_loop(n):
    total = 0
    for i in range(n):
        total += i
    return total


@micropython.viper
def portio(addr: int) -> int:
    return int(0)


def read_pin():
    import machine

    pin = machine.Pin(2, machine.Pin.IN)
    return pin.value()


@micropython.asm_viper
def add1(r0: int) -> int:
    return r0 + 1


@micropython.asm_thumb
def add2(r0: int) -> int:
    mov(r0, 3)
    bx(lr)
