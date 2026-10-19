export type FileEntityRef = { type: string; id: string };

export type FileMeta = {
  id: string;
  name: string;
  size: number;
  contentType: string;
  entity: FileEntityRef | null;
  createdAt: string;
};

export type {
  UploadedFile as FileDto,
  FilePage as FilePageDto,
} from '@starter/contracts';
