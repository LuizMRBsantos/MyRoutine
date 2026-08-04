import { Database } from '@nozbe/watermelondb';
import SQLiteAdapter from '@nozbe/watermelondb/adapters/sqlite';
import schema from './schema';
import Note from './models/Note';
import Transaction from './models/Transaction';
import HealthRecord from './models/HealthRecord';

const adapter = new SQLiteAdapter({
  schema,
  jsi: true, /* recommended for performance, requires prebuild */
  onSetUpError: error => {
    console.error("Database setup failed", error);
  }
});

export const database = new Database({
  adapter,
  modelClasses: [
    Note,
    Transaction,
    HealthRecord,
  ],
});
