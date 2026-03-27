import { useState, useEffect } from 'react';

interface UserProps {
  name: string;
  age: number;
  email?: string;
}

interface ApiResponse<T> {
  data: T;
  error?: string;
}

export function UserCard({ name, age, email }: UserProps) {
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    console.log('mounted');
  }, []);

  return expanded ? `${name} (${age})` : name;
}

export class UserService {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async getUser(id: string): Promise<ApiResponse<UserProps>> {
    const res = await fetch(`${this.baseUrl}/users/${id}`);
    return res.json();
  }

  async deleteUser(id: string): Promise<void> {
    await fetch(`${this.baseUrl}/users/${id}`, { method: 'DELETE' });
  }

  async listUsers(): Promise<ApiResponse<UserProps[]>> {
    const res = await fetch(`${this.baseUrl}/users`);
    return res.json();
  }
}

export const MAX_RETRIES = 3;
export const API_VERSION = 'v2';
