import { PersistedState } from 'runed';

// Per-admin preference that opens every Advanced disclosure; it lives in this browser until it moves to the user's settings
export const showAdvanced = new PersistedState('show-advanced-settings', false);
