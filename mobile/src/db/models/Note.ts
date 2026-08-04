import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, children, text } from '@nozbe/watermelondb/decorators';

export default class Note extends Model {
  static table = 'notes';

  @text('date') date!: string;
  @text('content') content!: string;
  @text('metadata') metadata?: string;
  @field('is_collection') isCollection!: boolean;

  @readonly @date('created_at') createdAt!: Date;
  @readonly @date('updated_at') updatedAt!: Date;

  @children('transactions') transactions!: any;
  @children('health_records') healthRecords!: any;
}
