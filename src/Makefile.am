
AM_CFLAGS = -g -Wall
LDFLAGS = -no-undefined $(GLIB_LIBS)
INCLUDES = $(GLIB_CFLAGS)

SUBDIRS = sysstat

dist_bin_SCRIPTS = perfmonger perfmonger-plot

noinst_LTLIBRARIES = libperfmonger.la
bin_PROGRAMS = perfmonger-record

libperfmonger_la_SOURCES = perfmonger.c perfmonger.h
libperfmonger_la_LIBADD = ./sysstat/libsysstat.la

perfmonger_record_SOURCES = main.c
perfmonger_record_LDADD = libperfmonger.la
