import React from 'react';
import {Composition, registerRoot} from 'remotion';
import {Root} from './Root';

registerRoot(() => {
  return (
    <Composition
      id="MqPromo"
      component={Root}
      durationInFrames={630}
      fps={30}
      width={1920}
      height={1080}
    />
  );
});
