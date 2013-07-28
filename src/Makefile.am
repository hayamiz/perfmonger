
AM_CFLAGS = -g -Wall
LDFLAGS = -no-undefined $(GLIB_LIBS)
INCLUDES = $(GLIB_CFLAGS)

SUBDIRS = sysstat ruby

bin_SCRIPTS = perfmonger

noinst_LTLIBRARIES = libperfmonger.la
bin_PROGRAMS = perfmonger-record

libperfmonger_la_SOURCES = libperfmonger.c perfmonger.h
libperfmonger_la_LIBADD = ./sysstat/libsysstat.la

perfmonger_record_SOURCES = main.c
perfmonger_record_LDADD = libperfmonger.la
