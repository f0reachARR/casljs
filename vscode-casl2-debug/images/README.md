# Icon

To generate the PNG icon from the SVG:

```bash
# Using ImageMagick
convert icon.svg -resize 128x128 icon.png

# Or using Inkscape
inkscape icon.svg -w 128 -h 128 -o icon.png

# Or use an online converter
```

The icon.svg is provided. Convert it to icon.png before packaging the extension.
