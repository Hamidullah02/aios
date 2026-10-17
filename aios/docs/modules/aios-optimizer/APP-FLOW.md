# aios-optimizer — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-optimizer module.

## 2. Scope

The scope of this document is the aios-optimizer
module. The module is a library.

## 3. Reference Documents

- aios-optimizer PRD. See
  `docs/modules/aios-optimizer/PRD.md`.
- aios-optimizer SRS. See
  `docs/modules/aios-optimizer/SRS.md`.
- aios-optimizer TRD. See
  `docs/modules/aios-optimizer/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the router. |

## 5. User Flow

The aios-optimizer module has no user-facing flow.

## 6. System Flow

### 6.1 Route Flow

The route flow runs as follows.

1. The caller calls `strategy.route` with the task.
2. The strategy extracts the features.
3. The strategy calls the classifier.
4. The classifier returns the tier and the
   confidence.
5. The strategy calls the cost tracker.
6. The cost tracker returns the budget status.
7. The strategy applies the budget-aware check.
8. The strategy returns the routing decision.

### 6.2 Cascade Flow

The cascade flow runs as follows.

1. The strategy tries the weak tier.
2. The strategy computes the confidence.
3. When the confidence is below the threshold,
   the strategy escalates to the medium tier.
4. When the medium tier is below the threshold,
   the strategy escalates to the strong tier.
5. The strategy returns the final decision.

### 6.3 Feedback Flow

The feedback flow runs as follows.

1. The caller calls `feedback.record` with the
   result.
2. The feedback records the result quality.
3. The feedback updates the strategy.
4. When the auto-escalate flag is set and the
   quality is below the threshold, the feedback
   escalates the tier for similar tasks.

## 7. Error Flows

### 7.1 Budget Exhausted

The budget exhausted flow runs as follows.

1. The cost tracker exhausts the budget.
2. The tracker raises `BudgetExhausted`.
3. The caller catches the error.
4. The caller stops the task.

### 7.2 Unknown Tier

The unknown tier flow runs as follows.

1. The strategy returns a tier that is not in the
   registry.
2. The registry raises `UnknownTier`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Route Sequence

The route sequence has the steps below.

1. The caller calls `strategy.route`.
2. The strategy calls the classifier.
3. The strategy calls the cost tracker.
4. The strategy returns the decision.
