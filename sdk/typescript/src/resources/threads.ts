// sdk/typescript/src/resources/threads.ts
/**
 * Threads resource for managing conversation threads and messages.
 */

import { EbotClient } from '../client';
import {
  Thread,
  Message,
  CreateThreadInput,
  ListOptions,
  PaginatedResponse,
  MessageChunk,
} from '../types';

export class Threads {
  constructor(private client: EbotClient) {}

  /**
   * List all threads.
   * 
   * @param options - Pagination and filter options
   * @returns Paginated list of threads
   */
  async list(options?: ListOptions): Promise<PaginatedResponse<Thread>> {
    return this.client.get<PaginatedResponse<Thread>>('/api/threads', {
      params: options,
    });
  }

  /**
   * Get a specific thread by ID.
   * 
   * @param id - The thread ID
   * @returns The thread details
   */
  async get(id: string): Promise<Thread> {
    return this.client.get<Thread>(`/api/threads/${id}`);
  }

  /**
   * Create a new thread.
   * 
   * @param data - The thread configuration
   * @returns The created thread
   * 
   * @example
   * ```typescript
   * const thread = await client.threads.create({
   *   title: 'New Conversation',
   *   agentId: 'agent_123',
   * });
   * ```
   */
  async create(data: CreateThreadInput): Promise<Thread> {
    return this.client.post<Thread>('/api/threads', { json: data });
  }

  /**
   * Delete a thread.
   * 
   * @param id - The thread ID
   */
  async delete(id: string): Promise<void> {
    await this.client.delete(`/api/threads/${id}`);
  }

  /**
   * Get all messages in a thread.
   * 
   * @param threadId - The thread ID
   * @param options - Pagination options
   * @returns Paginated list of messages
   */
  async getMessages(
    threadId: string,
    options?: ListOptions
  ): Promise<PaginatedResponse<Message>> {
    return this.client.get<PaginatedResponse<Message>>(
      `/api/threads/${threadId}/messages`,
      { params: options }
    );
  }

  /**
   * Send a message to a thread and wait for the complete response.
   * 
   * @param threadId - The thread ID
   * @param content - The message content
   * @param options - Additional options
   * @returns The assistant's response message
   * 
   * @example
   * ```typescript
   * const response = await client.threads.sendMessage('thread_123', 'Hello!');
   * console.log(response.content);
   * ```
   */
  async sendMessage(
    threadId: string,
    content: string,
    options?: {
      attachments?: Array<{ type: string; url: string }>;
      metadata?: Record<string, string>;
    }
  ): Promise<Message> {
    return this.client.post<Message>(`/api/threads/${threadId}/messages`, {
      json: {
        content,
        ...options,
      },
    });
  }

  /**
   * Send a message and stream the response.
   * 
   * @param threadId - The thread ID
   * @param content - The message content
   * @param options - Additional options
   * @yields Message chunks as they arrive
   * 
   * @example
   * ```typescript
   * for await (const chunk of client.threads.streamMessage('thread_123', 'Tell me a story')) {
   *   process.stdout.write(chunk.content);
   * }
   * ```
   */
  async *streamMessage(
    threadId: string,
    content: string,
    options?: {
      attachments?: Array<{ type: string; url: string }>;
      metadata?: Record<string, string>;
    }
  ): AsyncIterable<MessageChunk> {
    const stream = this.client.streamSSE(`/api/threads/${threadId}/messages/stream`, {
      method: 'POST',
      json: {
        content,
        ...options,
      },
    });

    for await (const event of stream) {
      yield event as MessageChunk;
    }
  }

  /**
   * Get a specific message by ID.
   * 
   * @param threadId - The thread ID
   * @param messageId - The message ID
   * @returns The message details
   */
  async getMessage(threadId: string, messageId: string): Promise<Message> {
    return this.client.get<Message>(
      `/api/threads/${threadId}/messages/${messageId}`
    );
  }

  /**
   * Regenerate the last assistant response.
   * 
   * @param threadId - The thread ID
   * @returns The regenerated message
   */
  async regenerate(threadId: string): Promise<Message> {
    return this.client.post<Message>(`/api/threads/${threadId}/regenerate`);
  }

  /**
   * Fork a thread from a specific message.
   * 
   * @param threadId - The source thread ID
   * @param messageId - The message to fork from
   * @returns The new forked thread
   */
  async fork(threadId: string, messageId: string): Promise<Thread> {
    return this.client.post<Thread>(`/api/threads/${threadId}/fork`, {
      json: { message_id: messageId },
    });
  }

  /**
   * Get thread summary/title.
   * 
   * @param threadId - The thread ID
   * @returns Generated summary
   */
  async summarize(threadId: string): Promise<{ title: string; summary: string }> {
    return this.client.post(`/api/threads/${threadId}/summarize`);
  }

  /**
   * Archive a thread.
   * 
   * @param threadId - The thread ID
   * @returns The archived thread
   */
  async archive(threadId: string): Promise<Thread> {
    return this.client.post<Thread>(`/api/threads/${threadId}/archive`);
  }

  /**
   * Unarchive a thread.
   * 
   * @param threadId - The thread ID
   * @returns The unarchived thread
   */
  async unarchive(threadId: string): Promise<Thread> {
    return this.client.post<Thread>(`/api/threads/${threadId}/unarchive`);
  }

  /**
   * Export thread as various formats.
   * 
   * @param threadId - The thread ID
   * @param format - Export format
   * @returns Export data
   */
  async export(
    threadId: string,
    format: 'json' | 'markdown' | 'html' = 'json'
  ): Promise<{ data: string; format: string }> {
    return this.client.get(`/api/threads/${threadId}/export`, {
      params: { format },
    });
  }
}
