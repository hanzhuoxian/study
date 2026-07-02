getpwnam:
	cc getpwnam.c -o getpwnam.app && ./getpwnam.app </etc/group
utsname:
	cc utsname.c -o utsname.app && ./utsname.app
clockgettime:
	cc clockgettime.c -o clockgettime.app && ./clockgettime.app
shadow:
	cc shadow.c -o shadow.app && sudo ./shadow.app
group:
	cc group.c -o group.app && ./group.app
utmp:
	cc utmp.c -o utmp.app && ./utmp.app
localtime:
	cc localtime.c -o localtime.app && ./localtime.app
gmtime:
	cc gmtime.c -o gmtime.app && ./gmtime.app
mktime:
	cc mktime.c -o mktime.app && ./mktime.app
strftime:
	cc strftime.c -o strftime.app && ./strftime.app
strptime:
	cc strptime.c -o strptime.app -D_GNU_SOURCE -D__USE_XOPEN && ./strptime.app
clean:
	rm *.app