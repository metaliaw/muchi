# Naming

- Functions Follow  
  **Verb + Noun + context** rhythm
- A Name longer than three words Suggests unclear Responsibility.
- A Variable that travels Holds three Words,  
  joined by its language.
- A Variable that lives in three Lines  
  Holds one word, because the scope says the rest.
- Three words Fit in memory and survive a rename.
- Sibling Functions Rhyme:  
  same Skeleton, different Word.
- One Idea Keeps one Word.  
  A Synonym Breaks the Rhyme.
- The Signature is the Bass Line.  
  If the Signature Grooves, the Body will Follow.
- A Construct Names the Responsibility, never the vendor.  
  *(Provisional)* `store` says what it Does;  
  `storegorm` says who it Called.
- A Filename may Name the Guest.  
  `store_gorm.go` Tells a reader where the ORM lives,  
  and no caller ever types it.
- A language with one type per file Loses that seam.  
  There the vendor Lives in the comment, and nowhere else.

```text
sumItemPrices      JavaScript, Go unexported
SumItemPrices      Go exported
sum_item_prices    Python
Sum Item Prices    Markdown, DeLaCase
```
