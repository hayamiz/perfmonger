
AM_CFLAGS = -g -Wall
LDFLAGS = -no-undefined $(GLIB_LIBS)
INCLUDES = $(GLIB_CFLAGS)

SUBDIRS = iostat

noinst_LTLIBRARIES = libperfmonger.la
bin_PROGRAMS = pgr

libperfmonger_la_SOURCES = perfmonger.c
libperfmonger_la_LIBADD = ./iostat/libiostat.la

pgr_SOURCES = main.c
pgr_LDADD = libperfmonger.la
