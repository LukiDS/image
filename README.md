![QOI Logo](https://qoiformat.org/qoi-logo.svg)

# QOI - The “Quite OK Image Format” for fast, lossless image compression

This is one of the golang versions of a En-/Decoder for the QOI - Format.

Be aware that this project is not optimized for speed or memory usage in any way, but it gets the job done.

The En-/Decoder DOES NOT load the complete image into memory. If speed is a concern, please save the image to a buffer before passing it to the En-/Decoder.

Examples may be provided at a later time.

Currently it only supports decoding qoi-files.
It will support encoding as well at a later time.

## More Infos

More infos at https://qoiformat.org
