#version 330

layout(location=0) in vec4 in_v;  // vertex coordinates
layout(location=2) in vec2 in_t;  // texture coordinates

uniform mat4  mvpm; // projection * model_view
out     vec2  t_uv; // pass uv coordinates through

void main() {
   gl_Position = mvpm * in_v;
   t_uv = in_t;
}
