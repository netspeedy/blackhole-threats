// Only `make website-capture` needs a browser, and it installs one itself (see the capture
// script in package.json), so installing the site's dependencies never downloads Chrome. npm 11
// already skips the download because package.json denies puppeteer's install script; this keeps
// npm 10, which CI uses, from running it.
module.exports = { skipDownload: true };
