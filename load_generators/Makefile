OBJECTS=
CFLAGS = -std=c99 -g -Wall -O3 -L/usr/local/lib -lsasl2 -lssl -lcrypto -lz -lmongoc-1.0 -lbson-1.0 -I/usr/local/include/libmongoc-1.0 -I/usr/local/include/libbson-1.0 
LDLIBS=
C=gcc

device_measurements: $(OBJECTS)
insert_simulation: $(OBJECTS)

clean:
	find . | egrep -v "Makefile|\.c$$|\.h$$|\.$$" | xargs rm -rf 
