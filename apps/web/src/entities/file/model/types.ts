export type FileMeta = {
  id: string;
  name: string;
  size: number;
  contentType: string;
  createdAt: string;
};

export type {
  UploadedFile as FileDto,
  FilePage as FilePageDto,
} from '@starter/contracts';
