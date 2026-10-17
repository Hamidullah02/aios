# ASD-STE100 Writing Standard — Project Application

## Purpose

This document states how the project applies ASD-STE100
Simplified Technical English (STE) Issue 9 to all prose text
in product, design, requirement, and test documents.

STE is a controlled natural language for technical
documentation. It makes text easier to read for non-native
English speakers and for readers without deep subject
knowledge. The project applies STE to all prose in:

- Product Requirements Documents (PRD)
- Software Requirements Specifications (SRS)
- Technical Requirements Documents (TRD)
- Application flow documents
- Wireframe annotations
- Implementation plans
- Code comments (limited subset, see below)
- User-facing text (CLI output, error messages, reports)

## Scope and Limit

STE is not a full style guide. The project uses STE
together with a structured technical writing layout
(defined in this document).

STE does not apply to:

- Programming language identifiers
- Configuration file values
- Standard log and protocol formats (JSON, YAML keys)
- Library and framework names
- Direct quotes from external specifications

## Core Rules

### Sentences

- Use a maximum of 20 words per sentence in procedures.
- Use a maximum of 25 words per sentence in descriptive text.
- Write one instruction per sentence.
- Write one topic per paragraph.
- Write a maximum of 6 sentences per paragraph.

### Voice and Tense

- Use the active voice. Example: "The orchestrator writes
  the audit row." Not: "The audit row is written by the
  orchestrator."
- Use the passive voice only when the agent is unknown.
  Example: "The data was corrupted." Not: "Something
  corrupted the data."
- Use the simple present tense for descriptive text.
- Use the imperative form for procedures.
  Example: "Run the test." Not: "You should run the test."

### Word Forms

- Use only these verb forms: infinitive, imperative,
  simple present, simple past, simple future, past
  participle (only as an adjective).
- Do not use the "-ing" form of a verb as a verb.
  Use the infinitive or the noun form.
  Example: write "The system performs the check."
  Not: "The system is performing the check."
- Do not omit sentence parts to shorten text.
  Keep the subject, verb, and object.

### Vocabulary

- Use only the approved STE words. See the dictionary
  in section "Approved Word List" of this document.
- For each unapproved word, use the approved alternative
  listed in the STE dictionary.
- Keep multi-word nouns to a maximum of three words.
  Example: "audit log" (2 words) is approved.
  "audit log rotation policy" (4 words) is not.
  Use "rotation policy for the audit log" instead.

### Lists

- Use vertical lists for complex or parallel content.
- Use numbered lists for procedures.
- Use bulleted lists for descriptive items.

### Safety and Warning Text

- Start each safety instruction with a clear command or
  condition. Example: "DO NOT run the destructive command."
- Place safety text before the action it warns about.

## Approved Word List (Project Subset)

The project uses the full STE dictionary. This list
shows the words the project uses most.

### Approved Verbs

- ACCEPT, ACCESS, ADD, ADJUST, ALLOW, ANALYZE, APPLY,
  APPROVE, ASK, ATTACH, ATTEMPT, BLOCK, BUILD, CALL,
  CANCEL, CHANGE, CHECK, CHOOSE, CLEAR, CLOSE, COLLECT,
  COMMIT, COMPARE, COMPLETE, CONNECT, CONTAIN, CONTINUE,
  CONVERT, COPY, CREATE, DECIDE, DECREASE, DEFINE, DELETE,
  DELIVER, DENY, DESCRIBE, DESIGN, DETERMINE, DISABLE,
  DISCONNECT, DISPLAY, DO, DOWNLOAD, DRAW, EDIT, ENABLE,
  ENSURE, ENTER, EXAMINE, EXECUTE, EXIT, EXPAND, EXPECT,
  EXPLAIN, FAIL, FETCH, FIND, FINISH, FIX, FOLLOW, FORMAT,
  GENERATE, GET, GIVE, GO, HANDLE, HOLD, IDENTIFY,
  IGNORE, INCLUDE, INCREASE, INDICATE, INPUT, INSERT,
  INSTALL, INSTRUCT, INTEGRATE, INTERRUPT, ISOLATE, JOIN,
  KEEP, KNOW, LIMIT, LIST, LOAD, LOCK, LOG, LOOK, MAKE,
  MANAGE, MARK, MATCH, MEASURE, MERGE, MODIFY, MONITOR,
  MOVE, NAME, NOTE, NOTIFY, OBSERVE, OFFER, OPEN, OPERATE,
  ORDER, OUTPUT, PASS, PAUSE, PERFORM, PERMIT, PLACE,
  PLAN, POINT, POST, PREDICT, PREPARE, PRESENT, PREVENT,
  PRINT, PROCEED, PROCESS, PRODUCE, PROMPT, PROPOSE,
  PROTECT, PROVIDE, PUBLISH, PULL, PUSH, PUT, READ, READY,
  RECEIVE, RECORD, REDUCE, REGISTER, REJECT, RELATE, RELEASE,
  REMOVE, RENAME, REPEAT, REPLACE, REPORT, REQUEST, REQUIRE,
  RESET, RESOLVE, RESTART, RESTORE, RESUME, RETURN, REVERT,
  REVIEW, REVOKE, RUN, SAVE, SEARCH, SELECT, SEND, SEPARATE,
  SERVE, SET, SHOW, SIGN, SORT, SPECIFY, SPLIT, START, STOP,
  STORE, SUBMIT, SUBSTITUTE, SUCCEED, SUPPLY, SUPPORT, SWITCH,
  TAKE, TELL, TERMINATE, TEST, TRACE, TRACK, TRANSFER,
  TRANSITION, TRANSLATE, TRIM, TURN, UNDO, UNLOCK, UPDATE,
  UPGRADE, USE, VALIDATE, VERIFY, VIEW, WAIT, WARN, WRITE

### Approved Nouns

- ACCESS, ACCOUNT, ACTION, ADAPTER, ADDRESS, AGENT, ALERT,
  ALLOWANCE, AMOUNT, ANSWER, API, AREA, ARGUMENT, ASSEMBLY,
  ATTEMPT, AUDIT, AUTH, BACKUP, BEHAVIOR, BLOCK, BODY,
  BOUNDARY, BRANCH, BREAKPOINT, BUDGET, BUFFER, BUILD, BUS,
  CALL, CAP, CAPABILITY, CAPACITY, CAUSE, CHANGE, CHECK, CLASS,
  CLIENT, CODE, COLUMN, COMMAND, COMMENT, COMthe standard `image` packageER, COMPONENT,
  CONDITION, CONFIG, CONNECTION, CONTENT, CONTEXT, CONTRACT,
  CONTROL, COST, COUNT, CPU, CRITERION, CYCLE, DATA, DATABASE,
  DEBUG, DECISION, DEFAULTS, DELAY, DEPENDENCY, DESIGN, DETAIL,
  DEVICE, DIRECTORY, DISK, DOCUMENT, DURATION, EDGE, EFFECT,
  ELEMENT, ENDPOINT, ENTRY, ENVELOPE, ENVIRONMENT, ERROR, EVENT,
  EVIDENCE, EXCEPTION, EXECUTION, EXIT, EXPECTATION, EXPENSE,
  EXPORT, EXTENSION, FACT, FAILURE, FEATURE, FIELD, FILE, FILTER,
  FLAG, FLOW, FORMAT, FORM, FRAMEWORK, FUNCTION, GAP, GATEWAY,
  GRAPH, GROUP, HANDLER, HEADER, HOST, HOUR, ID, IDLE, IMAGE,
  INDEX, INFO, INPUT, INSTALL, INSTANCE, INSTRUCTION, INTENT,
  INTERFACE, INTERVAL, ITEM, JOB, JSON, KEY, KILOBYTE, LABEL,
  LAG, LAYER, LEVEL, LIMIT, LINE, LINK, LIST, LOAD, LOCATION,
  LOCK, LOG, LOOP, MACHINE, MANAGER, MAP, MARK, MATCH, MAXIMUM,
  MEAN, MEASURE, MEMBER, MEMORY, MESSAGE, METADATA, METHOD,
  MINIMUM, MINUTE, MODE, MODEL, MODULE, NAME, NETWORK, NODE,
  NOTE, NUMBER, OBJECT, OBSERVATION, OPERATION, OPTION, ORDER,
  OUTPUT, OWNER, PACKAGE, PARAMETER, PART, PATH, PAYLOAD, PEAK,
  PERMISSION, PHASE, PIPE, PLACE, PLAN, PLATFORM, POINT, POOL,
  PORT, POSITION, PREFIX, PROCESS, PROFILE, PROOF, PROTOCOL,
  PROXY, QUERY, QUEUE, RANGE, RATE, REASON, RECORD, REDACTION,
  REFERENCE, REGION, REGISTRY, RELATION, REPORT, REQUEST, RESET,
  RESOURCE, RESPONSE, RESULT, RETRY, REVIEW, ROLE, ROW, RULE,
  RUN, RUNNER, SCOPE, SCREEN, SCRIPT, SECTION, SELECTION,
  SENDER, SESSION, SET, SETTING, SIGNAL, SITE, SIZE, SLOT, SOURCE,
  SPACE, SPEC, STAGE, STATE, STATEMENT, STATUS, STEP, STORE,
  STRING, STRUCTURE, SUBJECT, SUBSYSTEM, SUMMARY, SUPPLIER, SWITCH,
  SYMBOL, TABLE, TAG, TARGET, TASK, TEMPLATE, TEST, THRESHOLD,
  TICKET, TIME, TIMEOUT, TIMER, TOKEN, TOOL, TOPIC, TRACE, TRACK,
  TRAFFIC, TRANSITION, TRIAL, TRIGGER, TYPE, UNIT, UPDATE, USER,
  VALIDATION, VALUE, VARIABLE, VERSION, VOLUME, WARNING, WORKER,
  WORKSPACE, YEAR, ZONE

### Approved Adjectives

- ABLE, ACCEPTABLE, ACCESSIBLE, ACTIVE, ADDITIONAL, APPLICABLE,
  APPROVED, AVAILABLE, AVERAGE, BAD, BASIC, BUSY, CAPABLE, CHOSEN,
  CLEAR, CLOSED, COMMON, COMPLETE, CONNECTED, CORRECT, CURRENT,
  DANGEROUS, DEFAULT, DEPENDENT, DIRECT, DISABLED, DISCONNECTED,
  EMPTY, ENABLED, EQUAL, EXACT, EXPECTED, EXTERNAL, FALSE, FAST,
  FINAL, FIXED, FREE, FULL, GLOBAL, GOOD, HARD, HIGH, IDLE, INACTIVE,
  INTERNAL, INVALID, LEGAL, LOCAL, LOCKED, LOW, MAIN, MAXIMUM, MINIMUM,
  MODULE, NEW, NEXT, NORMAL, OPEN, OPTIONAL, ORIGINAL, OTHER, OVERALL,
  PASSED, PENDING, PERMANENT, PERSONAL, PHYSICAL, PLANNED, POSSIBLE,
  PRESENT, PREVIOUS, PRIVATE, PUBLIC, READY, REAL, RECEIVED, REQUIRED,
  REVERSED, RUNNING, SAFE, SAME, SECOND, SELECTED, SEPARATE, SET,
  SIGNED, SIMPLE, SINGLE, SLOW, SPECIFIC, STABLE, STANDARD, STATIC,
  SUCCESSFUL, SUPPORTED, TEMPORARY, TERMINAL, TESTED, TOTAL, UNKNOWN,
  UNSUPPORTED, UNUSED, UNVERIFIED, UNLOCKED, UNSAFE, USED, USEFUL,
  VALID, VERIFIED, VISIBLE, WRONG, ZERO

### Approved Adverbs

- ABOUT, ABOVE, AFTER, AGAIN, ALSO, ANY, AROUND, AS, BACK, BELOW,
  BETWEEN, BOTH, DOWN, EVEN, EVER, HERE, HOWEVER, IMMEDIATELY, INSIDE,
  INSTEAD, JUST, LATER, LESS, MANUALLY, MORE, NEVER, NEXT, NOT, NOW,
  OFTEN, ONLY, OUTSIDE, OVER, PER, QUICKLY, RECENTLY, REGULARLY,
  SLOWLY, SO, STILL, THEN, THERE, TODAY, TOGETHER, TOO, UNDERNEATH,
  UP, VERY, WHEN, WHERE, WHY, YES, YESTERDAY, YET

### Approved Prepositions

- ABOUT, ABOVE, ACROSS, AFTER, AGAINST, ALONG, AMONG, AROUND, AT,
  BEFORE, BEHIND, BELOW, BENEATH, BESIDE, BETWEEN, BY, DOWN, DURING,
  EXCEPT, FOR, FROM, IN, INSIDE, INTO, NEAR, OF, OFF, ON, ONTO, OUT,
  OUTSIDE, OVER, PAST, SINCE, THROUGH, THROUGHOUT, TILL, TO, TOWARD,
  UNDER, UNDERNEATH, UNTIL, UP, UPON, WITH, WITHIN, WITHOUT

### Approved Conjunctions

- AND, BOTH, BUT, EITHER, IF, NEITHER, OR, THEN, WHEN, WHERE, WHETHER,
  WHILE, WHY

### Approved Pronouns

- HE, HER, HIM, HIS, I, IT, ITS, ME, MY, OUR, SHE, THEIR, THEM, THESE,
  THEY, THIS, THOSE, WE, WHAT, WHICH, WHO, WHOM, WHOSE, YOU, YOUR

## Forbidden Words and Approved Alternatives

The project uses these common replacements. The list
shows words to avoid and the approved alternative.

| Forbidden | Approved Alternative |
| --- | --- |
| acceptance (n) | ACCEPT (v) |
| accessible (adj) | ACCESSIBLE is approved |
| accommodate (v) | HOLD (v) |
| accomplish (v) | COMPLETE (v) |
| according to | PER (prep) |
| additional | ADDITIONAL is approved |
| address (a problem) | SOLVE (v) |
| advise (v) | TELL (v) |
| afford (v) | GIVE (v) |
| aforementioned | PREVIOUS is approved |
| aggregate (v) | COMBINE (v) |
| allocate (v) | ASSIGN (v) |
| allow (v) | ALLOW is approved |
| alternate (v) | SWITCH (v) |
| anticipate (v) | EXPECT (v) |
| apparent (adj) | CLEAR is approved |
| appreciable (adj) | MEASURABLE is approved |
| apprise (v) | TELL (v) |
| appropriate (adj) | SUITABLE is approved |
| approximate (v) | APPROXIMATE is approved |
| assist (v) | HELP (v) |
| attain (v) | GET (v) |
| attempt (v) | ATTEMPT is approved |
| author (n) | WRITER (n) |
| authorize (v) | APPROVE (v) |
| begin (v) | START (v) |
| beneficial (adj) | USEFUL is approved |
| capability (n) | CAPABILITY is approved |
| categorize (v) | CLASSIFY (v) |
| cease (v) | STOP (v) |
| close (a deal) | COMPLETE (v) |
| commence (v) | START (v) |
| communicate (v) | TELL (v) |
| compensate (v) | PAY (v) |
| complete (v) | COMPLETE is approved |
| component (n) | COMPONENT is approved |
| comprise (v) | INCLUDE (v) |
| concerning (prep) | ABOUT (prep) |
| consequently (adv) | THEN (adv) |
| consolidate (v) | COMBINE (v) |
| constitutes (v) | IS (v) |
| contains (v) | CONTAIN is approved |
| demonstrate (v) | SHOW (v) |
| depart (v) | LEAVE (v) |
| deprecate (v) | STOP (v) use |
| desire (v) | WANT (v) |
| determine (v) | DETERMINE is approved |
| disclose (v) | SHOW (v) |
| discontinue (v) | STOP (v) |
| display (v) | DISPLAY is approved |
| disseminate (v) | SEND (v) |
| documentation (n) | DOCUMENT is approved |
| during (prep) | DURING is approved |
| employ (v) | USE (v) |
| enable (v) | ENABLE is approved |
| enclose (v) | INCLUDE (v) |
| encounter (v) | MEET (v) |
| endeavor (v) | TRY (v) |
| ensure (v) | ENSURE is approved |
| enumerate (v) | LIST (v) |
| erroneous (adj) | WRONG is approved |
| essential (adj) | REQUIRED is approved |
| establish (v) | SET (v) |
| evaluate (v) | TEST (v) |
| examine (v) | EXAMINE is approved |
| facilitate (v) | HELP (v) |
| finalise (v) | COMPLETE (v) |
| following (adj) | NEXT is approved |
| frequently (adv) | OFTEN (adv) |
| generate (v) | GENERATE is approved |
| identical (adj) | SAME is approved |
| implement (v) | USE (v) |
| imply (v) | MEAN (v) |
| inception (n) | START (n) |
| incorporate (v) | INCLUDE (v) |
| indicate (v) | INDICATE is approved |
| inform (v) | TELL (v) |
| initiate (v) | START (v) |
| inquire (v) | ASK (v) |
| inspect (v) | EXAMINE (v) |
| instantiate (v) | CREATE (v) |
| intend (v) | PLAN (v) |
| interact (v) | USE (v) |
| interpret (v) | READ (v) |
| leverage (v) | USE (v) |
| locate (v) | FIND (v) |
| magnitude (n) | SIZE (n) |
| maintain (v) | KEEP (v) |
| methodology (n) | METHOD (n) |
| modify (v) | MODIFY is approved |
| monitor (v) | MONITOR is approved |
| necessitate (v) | REQUIRE (v) |
| notify (v) | NOTIFY is approved |
| numerous (adj) | MANY is approved |
| objective (n) | TARGET (n) |
| observe (v) | OBSERVE is approved |
| obtain (v) | GET (v) |
| occupy (v) | FILL (v) |
| occur (v) | HAPPEN (v) |
| optimum (adj) | BEST is approved |
| perform (v) | PERFORM is approved |
| permit (v) | PERMIT is approved |
| possess (v) | HAVE (v) |
| prefer (v) | CHOOSE (v) |
| presently (adv) | NOW (adv) |
| previous (adj) | PREVIOUS is approved |
| primarily (adv) | MAINLY (adv) |
| prioritize (v) | ORDER (v) |
| procure (v) | GET (v) |
| prohibit (v) | PREVENT (v) |
| provide (v) | PROVIDE is approved |
| purchase (v) | BUY (v) |
| receive (v) | RECEIVE is approved |
| recommend (v) | ADVISE (v) |
| record (n) | RECORD is approved |
| reduce (v) | REDUCE is approved |
| refer (v) | POINT (v) |
| regarding (prep) | ABOUT (prep) |
| register (v) | REGISTER is approved |
| regulate (v) | CONTROL (v) |
| reimburse (v) | PAY (v) |
| reiterate (v) | REPEAT (v) |
| related (adj) | CONNECTED is approved |
| release (v) | RELEASE is approved |
| relinquish (v) | GIVE (v) |
| remain (v) | STAY (v) |
| remove (v) | REMOVE is approved |
| render (v) | MAKE (v) |
| represent (v) | SHOW (v) |
| request (v) | REQUEST is approved |
| require (v) | REQUIRE is approved |
| rescind (v) | CANCEL (v) |
| resolve (v) | RESOLVE is approved |
| respectively (adv) | drop the word |
| restriction (n) | LIMIT (n) |
| retain (v) | KEEP (v) |
| retrieve (v) | GET (v) |
| satisfy (v) | MEET (v) |
| select (v) | SELECT is approved |
| several (adj) | SOME is approved |
| similar (adj) | SAME is approved |
| solicit (v) | ASK (v) |
| specify (v) | SPECIFY is approved |
| subsequent (adj) | NEXT is approved |
| substantial (adj) | LARGE is approved |
| sufficient (adj) | ENOUGH is approved |
| terminate (v) | TERMINATE is approved |
| transfer (v) | TRANSFER is approved |
| transmit (v) | SEND (v) |
| utilise (v) | USE (v) |
| validate (v) | VALIDATE is approved |
| verify (v) | VERIFY is approved |
| whereas (conj) | BUT (conj) |
| whereby (conj) | drop the word |
| whilst (conj) | WHILE (conj) |
| with regard to | ABOUT (prep) |
| with respect to | ABOUT (prep) |

## Project Document Structure

The project organizes each document with a fixed layout.
The layout makes the document easier to read and
translate.

### Top-Level Sections

1. Purpose — one sentence about the document.
2. Scope — what is in the document and what is not.
3. Reference Documents — other documents this one depends on.
4. Definitions — words with special meaning in the document.
5. Content — the body of the document, in numbered sections.

### Numbering

- Use one-level numbering for short documents (1, 2, 3).
- Use two-level numbering for longer documents (1.1, 1.2, 2.1).
- Use three-level numbering only for documents with
  more than 20 sections.

### Tables

- Use tables for parallel data.
- Each table has a title above the table.
- Each column has a header.
- Use the first column for the identifier or name.

### Lists

- Use a numbered list for sequential steps.
- Use a bulleted list for parallel items.
- Start each item with a capital letter.

### Cross-References

- Reference another document by name and section number.
  Example: "See SRS, section 3.2."
- Do not use phrases like "as stated above" or
  "as mentioned earlier."

## Procedures (How-To Text)

Use this format for all step-by-step procedures.

1. Start each step with an imperative verb.
2. Keep each step to a maximum of 20 words.
3. Use one step per action.
4. Place a safety warning before the action it warns about.
5. End the procedure with a verification step.

## Description Text

Use this format for background and reference text.

- Use simple present tense.
- Keep sentences to a maximum of 25 words.
- Keep paragraphs to a maximum of 6 sentences.
- Use one topic per paragraph.

## Review Checklist

The project reviews every document against this checklist
before the document is approved.

- [ ] Each sentence is 20 words or fewer (procedures) or
      25 words or fewer (descriptions).
- [ ] Each paragraph has 6 sentences or fewer.
- [ ] Each paragraph has one topic.
- [ ] The active voice is used. The passive voice is used
      only when the agent is unknown.
- [ ] Verb forms are limited to the approved list.
- [ ] The "-ing" form is not used as a verb.
- [ ] Multi-word nouns have 3 words or fewer.
- [ ] All vocabulary is from the approved word list.
      Replacements are applied for forbidden words.
- [ ] Safety instructions start with a clear command.
- [ ] Lists use vertical layout.
- [ ] Tables have a title and column headers.
- [ ] Cross-references use the document name and section
      number.

## Code Comments

Code comments are a limited form of technical writing.
The project applies STE to comments with these limits.

- Function docstrings follow the project docstring
  standard. See the Coding Standard for the format.
- Inline comments explain why, not what.
- A comment is one line of text, or two lines of text.
- A comment does not use sentence fragments.
- A comment does not use the future tense.

## Reference

- ASD-STE100 Simplified Technical English, Issue 9,
  January 15, 2025. Aerospace, Security and Defence
  Industries Association of Europe, Brussels.
- Wikipedia, "Simplified Technical English",
  https://en.wikipedia.org/wiki/ASD-STE100
- ASD-STE100 home page, https://www.asd-ste100.org/
