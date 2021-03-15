# Bugs

## Race conditions

runj does not currently protect against race conditions when starting jails.  If
two `runj start` invocations happen concurrently for the same jail, the behavior
is undefined.

## Garbage collection

runj can fail to clean up the state directory it creates for a jail, leading to
conflicts when attempting to start another jail with the same name.
