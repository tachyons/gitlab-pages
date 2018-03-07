# MIME DB

This Go package uses generators to convert [this database](https://github.com/jshttp/mime-db)
into additions to the stdlib `mime` package.

Since all the work is done at compile time, the MIME types end up embedded in
the binary, loading them on startup is fast, and you still get sensible results
when `/etc/mime.types` is unavailable on your platform!

This work is somewhat inspired by [mime-ext-go](https://github.com/mytrile/mime-ext-go),
which lacks the automatic generation (and so easy update) to be found in this
package.

The version of the mime-db package used is tracked in the VERSION file, and
updates will be given a corresponding tag.
