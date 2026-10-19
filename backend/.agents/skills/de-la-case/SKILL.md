---
name: de-la-case
description: Convert code identifiers or prose into a DeLaCase suggestion — identify important entities and their interactions, with up to three emphasis capitals per passage between punctuation marks; the grammatical initial is free. For identifiers, apply the language's case convention plus Verb+Noun+context (≤3 words). Use when the user asks to "DeLaCase" a name, sentence, or identifier, or to rewrite something in the manifesto's convention.
---

# DeLaCase

A Converter, not a Checklist. It takes Code or Words  
and Returns one Suggestion, plus one line naming the relationship.

The checklist is [OneTwoRefactor](../one-two-refactor/SKILL.md).  
Use that while writing a whole Unit. Use this to rename one.  
To hear the pattern under an explanation, use the  
[Dove](../../agents/dove.md) agent.

The Canon lives in [DeLaCase](../../canon/rules/de-la-case.md).

## How to Convert

1. Classify the Input — prose or code?  
   Prose is a sentence or a comment.  
   Code is an identifier, a function, a variable.

2. Split the prose at punctuation that separates passages.  
   Commas, semicolons, colons and sentence endings renew the Budget.  
   Follow the canon for clause-delimiting dashes, parentheses and line breaks.
   Punctuation within a word or number does not split the passage.

3. Keep the grammatical initial capitalized without spending Budget.  
   A word after a comma has no free capital merely because it starts a passage.
   Proper names and acronyms retain their established spelling outside the budget.

4. Identify the important Entities and their Interactions.  
   Capitalize up to three words per passage that make the relationship visible.
   Two entities and one interaction are a useful shape, not a noun-and-verb quota.
   Choose by meaning rather than grammatical voice or fixed word spacing.
   Use fewer when the phrase needs fewer; never invent a participant.

5. Lowercase other discretionary emphasis.  
   A list of names stays exempt, and punctuation must not be added for budget.

6. For code, apply the language Convention —  
   PascalCase when exported,  
   camelCase when unexported,  
   snake_case for Python.  
   Then fit the name to Verb + Noun + context, three words at most.

7. Return the Suggestion, and one line naming the Entities or Interaction it highlights.

## Examples

### Prose — entities and interaction

Input: "the person guides the machine"  
Output: "The Person Guides the Machine."\
Why: *Person* and *Machine* are the entities; *Guides* names their interaction.
The initial *The* is free, leaving three spent capitals.

### Prose — fewer are Enough

Input: "the person rests"  
Output: "The Person Rests."\
Why: One entity and its action carry the claim without another participant.

### Prose — punctuation

Input: "the person guides the machine, the machine supports the person"  
Output: "The Person Guides the Machine, the Machine Supports the Person."\
Why: Each passage highlights its relationship within a separate budget.
Only the sentence's grammatical initial is free.

### Prose — definition

Input: "rest is care"  
Output: "Rest is Care."\
Why: The definition connects two concepts; *Rest* already has its free initial.

### Prose — imperative

Input: "protect your attention"  
Output: "Protect your Attention."\
Why: The action already has its free initial; the emphasis identifies its entity.

### Prose — bold

Input:  "spend the capital where you would raise your voice"  
Output: "**Spend the capital where you would raise your voice.**"  
Why:    Bold is the second Tier, one to a section, never on a word a capital already marks.

### Code — Go exported

Input:  GetUserAccountDataById  
Output: FetchAccount  
Why:    Fetch is the Action, account the entity; context and id were local, so three words held.

### Code — Go unexported

Input:  getUserAccountDataById  
Output: fetchAccount  
Why:    camelCase Stays inside; the same three words, lowercased at the front.

### Code — Python

Input:  get_user_account_data_by_id  
Output: fetch_account  
Why:    snake_case for Python; three words again, id and data were local to the scope.

## When not to Convert

- A name that already Reads as three words in its language.
- A variable that lives in three lines — it may keep one Word,  
  because the scope already says the rest.
- An enumeration of names, a table Column or a diagram label.  
  None of them is a Sentence, so none of them spends.
- An identifier Spends nothing, because it is not a sentence.  
  Case there belongs to the Language, and the language decides.
- Final text shown to users in an interface uses ordinary capitalization.\
  Use DeLaCase there only when explicitly requested.
