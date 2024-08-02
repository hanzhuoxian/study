stat:
	cc stat.c -o stat.app && ./stat.app stat.c /etc/passwd /etc /dev/log \
  		/dev/tty /dev/sr0 /dev/cdrom /etc/passwd  && rm -rf stat.app
access:
	cc access.c -o access.app && ./access.app /etc/passwd && rm -rf access.app
UMASK_FILE := bar foo haha
umask:
	cc umask.c -o umask.app -l apue && ./umask.app /etc/passwd && rm -rf umask.app && ls  -al $(UMASK_FILE) && rm $(UMASK_FILE)
getcwd:
	cc getcwd.c -l apue -o getcwd.app && ./getcwd.app
rdev:
	cc rdev.c -l apue -o rdev.app && ./rdev.app /dev/ttyr0