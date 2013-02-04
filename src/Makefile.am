
AM_CFLAGS = -g -Wall
LDFLAGS = -no-undefined $(GLIB_LIBS)
INCLUDES = $(GLIB_CFLAGS)

SUBDIRS = sysstat

noinst_LTLIBRARIES = libperfmonger.la
bin_PROGRAMS = perfmonger

libperfmonger_la_SOURCES = perfmonger.c perfmonger.h
libperfmonger_la_LIBADD = ./sysstat/libsysstat.la

perfmonger_SOURCES = main.c
perfmonger_LDADD = libperfmonger.la

pgr: perfmonger
	cp $< $@
