layout(location=0) in vec4 in_v;  // vertex coordinates
layout(location=2) in vec2 in_t;  // texture coordinates

uniform mat4 pm;   // projection matrix
uniform mat4 vm;   // view matrix
uniform mat4 mm;   // model matrix
out     vec2  v_t; // pass uv coordinates through

void main() {
   v_t = in_t;
   gl_Position = pm * vm * mm * in_v;
}
