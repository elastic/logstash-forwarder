VERSION=1.4.6
URL=http://www.globalish.com/am//apr/apr-${VERSION}.tar.gz
TARBALL=$(basename $URL)
WORKDIR=apr-${VERSION}

workdir() {
  (cd $WORKDIR; "$@")
}

[ ! -f $TARBALL ] && wget -O $TARBALL $URL
[ ! -d $WORKDIR ] && tar -zxf $TARBALL
[ ! -f $WORKDIR/config.log ] && workdir ./configure --prefix=$PREFIX
[ ! -f $WORKDIR/.libs/libapr-1.so.0.4.6 ] && workdir make
[ ! -f $PREFIX/lib/libapr-1.so.0.4.6 ] && workdir make install

