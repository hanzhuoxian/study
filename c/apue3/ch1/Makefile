OBJS := $(patsubst %.c,%.o,$(wildcard copy.c))
hello: $(OBJS)
	cc -o copy $(OBJS)
	@echo $^
hello.o: hello.c
pid: pid.o
	cc -o pid pid.o
pid.o: pid.c

gexec: gexec.o
	cc -o gexec gexec.o
gexec.o: gexec.c

clean:
	-rm *.o 
	@echo success
type: 
	cc type.c -o type.app && ./type.app && rm -rf type.app