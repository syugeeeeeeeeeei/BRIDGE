declare const process: any;
declare module "node:child_process" { export function spawn(command: string, args?: string[], options?: any): any; }
declare module "node:fs" { export const constants: any; export function accessSync(path: string, mode?: any): void; export function existsSync(path: string): boolean; }
declare module "node:path" { export const delimiter: string; export function dirname(path: string): string; export function join(...paths: string[]): string; }
declare module "node:url" { export function fileURLToPath(url: string | URL): string; }
