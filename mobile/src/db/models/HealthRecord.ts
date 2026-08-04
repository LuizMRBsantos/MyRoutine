import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, relation, text } from '@nozbe/watermelondb/decorators';
import type Note from './Note';

export default class HealthRecord extends Model {
  static table = 'health_records';

  @text('type') type!: string; // 'run', 'bike', 'swim', 'gym', 'wellness'
  @field('distance') distance?: number;
  @field('time') time?: number;
  @field('rpe') rpe?: number;
  @text('macros') macros?: string;
  @text('date') date!: string;

  @text('push_status') pushStatus!: string;
  @field('pushed_at') pushedAt?: number;

  @relation('notes', 'note_id') note!: Note;

  @readonly @date('created_at') createdAt!: Date;
  @readonly @date('updated_at') updatedAt!: Date;
}
