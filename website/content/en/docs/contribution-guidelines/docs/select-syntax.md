---
title: Selected Syntax
---

For the most part, the documentation should be written in simple
markdown. Common departures from vanilla markdown are explained here.

## Frontmatter

To be included in the built documentation, metadata must be included at
the top of every file. 

Minimally:

```
---
title: The Title
---
```

Other frequently used fields are: `linkTitle` (for navigation), `weight`
(for order), and `description`.

[More on
Frontmatter](https://www.docsy.dev/docs/adding-content/content/#page-frontmatter)

## Including Images

There are multiple ways to do this, but the simplest way is to put the
file in the `website/static` directory, and the image is included as-is
with standard markdown:

```
![What the user sees](/filename.png)
```
