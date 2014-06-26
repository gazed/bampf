#version 330

layout(location=0) in vec4 in_v;  // vertex coordinates
layout(location=2) in vec2 in_t;  // texture coordinates

uniform mat4  mvpm;  // projection * model_view
uniform vec3  scale; // scale
out     vec2  t_uv;  // pass uv coordinates through

void main() {
   mat4 bb = mvpm;
   bb[0][0] = 1.0;
   bb[1][0] = mvpm[1][0];
   bb[2][0] = 0.0;
   bb[0][1] = 0.0;
   bb[1][1] = 1.0;
   bb[2][1] = 0.0;
   bb[0][2] = 0.0;
   bb[1][2] = mvpm[1][2];
   bb[2][2] = 1.0;
   vec4 vpos = in_v;
   vpos.xyz = vpos.xyz * scale;
   gl_Position = bb * vpos;
   t_uv = in_t;
}
