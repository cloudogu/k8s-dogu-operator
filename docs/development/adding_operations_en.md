# Adding operations to the operator

If the custom resource is extended by fields which should lead to a new operation in the operator,
the operator must then be modified in the following places:
- `evaluateRequiredOperation()` or `appendRequiredPostInstallOperations()` must be adjusted so that the operation is recognized.
  It should be noted here that the operator can recognize multiple operations. Thus, the operation must be inserted at the appropriate place.
- In `executeRequiredOperation()` the operations are then evaluated and executed.
  Only the first recognized operation is executed. If there is more than one operation, a requeue is initiated to execute the other operations.
