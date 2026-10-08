declare module 'fake-async' {
  export function work(callback: () => void): void;
  export function load(): Promise<string>;
}
