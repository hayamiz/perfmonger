AC_PREREQ(2.59)

AC_INIT(perfmonger, 0.4.0, y.hayamizu@gmail.com)
AC_CONFIG_AUX_DIR([config])
AC_CONFIG_HEADER([config.h])
AC_CONFIG_MACRO_DIR([m4])

AM_INIT_AUTOMAKE($PACKAGE_NAME, $PACKAGE_VERSION)
AC_SUBST(PACKAGE_VERSION)

AC_PROG_LIBTOOL

AC_CONFIG_FILES([Makefile
	src/Makefile
	src/sysstat/Makefile
	src/ruby/Makefile
	src/ruby/perfmonger/Makefile
	src/ruby/perfmonger/command/Makefile
	data/Makefile
	data/assets/Makefile
	data/assets/js/Makefile
	data/assets/css/Makefile
	data/assets/img/Makefile
	test/Makefile])
AC_CONFIG_FILES([src/sysstat/sysconfig.h:src/sysstat/sysconfig.in
	src/ruby/perfmonger/version.rb
	src/ruby/perfmonger/config.rb])
AC_CONFIG_FILES([src/perfmonger], [chmod +x src/perfmonger])


dnl **************************************************************
dnl Check for GLib
dnl **************************************************************

PKG_CHECK_MODULES([GLIB],
                  [glib-2.0 gthread-2.0],
                  [ac_glib_2_0_available="yes"],
                  [ac_glib_2_0_available="no"])

if test "$ac_glib_2_0_available" = "no"; then
  AC_MSG_ERROR([GLib is required.])
fi
AM_CONDITIONAL([WITH_GLIB], [test "$ac_glib_2_0_available" = "yes"])

dnl **************************************************************
dnl Check for Ruby
dnl **************************************************************
ruby_available="no"
AC_ARG_WITH([ruby],
            AS_HELP_STRING([--with-ruby=PATH],
                           [Ruby interpreter path (default: auto-detect)]),
            [RUBY="$withval"])

if test "$RUBY" = "no"; then
  : # ignore
elif test "$RUBY" = ""; then
  AC_PATH_PROG(RUBY, ruby, none)
  if test "$RUBY" != "none"; then
    ruby_available="yes"
  fi
else
  AC_CHECK_FILE([$RUBY],
                [ruby_available="yes"],
                [AC_MSG_ERROR([$RUBY is not found.])])
fi

if test "x$ruby_available" != "xyes"; then
  AC_MSG_ERROR([Ruby is required.])
fi

perfmonger_ruby_path=$RUBY
AC_SUBST(perfmonger_ruby_path)

dnl **************************************************************
dnl Check for Cutter
dnl **************************************************************

m4_ifdef([AC_CHECK_GCUTTER],
         [AC_CHECK_GCUTTER],
         [cutter_use_cutter="no"])

AM_CONDITIONAL([WITH_CUTTER], [test "$cutter_use_cutter" != "no"])
if test "$cutter_use_cutter" = "yes"; then
  AC_DEFINE(WITH_CUTTER, 1, [Define to 1 if you use Cutter])
fi


dnl **************************************************************
dnl Check for path
dnl **************************************************************

if test "$exec_prefix" = "NONE"; then
  if test "$prefix" = "NONE"; then
    perfmonger_exec_prefix="$ac_default_prefix"
    perfmonger_prefix="$ac_default_prefix"
  else
    perfmonger_exec_prefix="$prefix"
    perfmonger_prefix="$prefix"
  fi
else
  perfmonger_exec_prefix="$exec_prefix"
fi


if echo -- "$bindir" | grep '\$' > /dev/null 2>&1; then
  perfmonger_bindir="$perfmonger_exec_prefix/bin"
else
  perfmonger_bindir="$bindir"
fi


if echo -- "$libdir" | grep '\$' > /dev/null 2>&1; then
  perfmonger_libdir="$perfmonger_exec_prefix/lib"
else
  perfmonger_libdir="$libdir"
fi
perfmonger_libdir="$perfmonger_libdir/perfmonger"

perfmonger_rubylibdir="$perfmonger_libdir/ruby"


if echo -- "$datarootdir" | grep '\$' > /dev/null 2>&1; then
  perfmonger_datarootdir="$perfmonger_prefix/share"
else
  perfmonger_datarootdir="$datarootdir"
fi
perfmonger_datarootdir="$perfmonger_datarootdir/perfmonger"


AC_SUBST(perfmonger_bindir)
AC_SUBST(perfmonger_rubylibdir)
AC_SUBST(perfmonger_datarootdir)


dnl **************************************************************
dnl Misc
dnl **************************************************************

perfmonger_ac_date=`date +'%Y-%m-%d %H:%M:%S'`
AC_SUBST(perfmonger_ac_date)

perfmonger_build_aux=""
if test -d '.git'; then
  perfmonger_build_aux=", git:$(git log | head -1 | awk '{print $2}' | head -c 8)"
fi

AC_SUBST(perfmonger_build_aux)

AC_OUTPUT


echo
echo "Configure Result:"
echo "== Prerequisites =="
echo "  Ruby : $RUBY"
echo " GLib2 : $ac_glib_2_0_available"
echo ""
echo "== Optional requirements =="
echo " Cutter(optional) : $CUTTER"
echo ""
echo
