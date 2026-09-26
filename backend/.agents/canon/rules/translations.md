# Translations

> Provisional. Written from practice, not yet Weathered.

- A Translation is the same Document, read in another tongue.  
  The Language changes the reading, never the Identity.
- English Keeps the bare name; the other adds its Code.  
  `layers.md` and `layers.es.md`, side by side.
- Each Pair Links the other near the title.  
  A reader should not have to Search.
- The README Indexes each language in its own File.  
  `README.md` and `README.es.md`.
- A Change of meaning Edits both halves in one commit.  
  A Pair that drifts is two documents, not one.
- A Link Stays inside its language.  
  English points to English; the `.es.md` points to its own.
- A Rename Moves both halves and every link to them.  
  A link into another repository is Checked there, not here.

The Check Runs in one Loop.  
Each English file may name its own Pair, and no other `.es.md`.

```sh
for f in $(git ls-files '*.md' | grep -v '\.es\.md$'); do
  grep -Hn ']([^)]*\.es\.md' "$f" | grep -v "$(basename "$f" .md)\.es\.md"
done
```

Any line it prints is a Pair already broken.
