import {Config} from '@remotion/cli/config';

Config.setVideoImageFormat('jpeg');
Config.setOverwriteOutput(true);
Config.setPixelFormat('yuv420p');
Config.setCodec('h264');
Config.setEntryPoint('./src/index.tsx');

export const config: Config = {
  codec: 'h264',
  fps: 30,
  proResProfile: undefined,
};
