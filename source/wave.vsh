#version 330

layout(location=0) in vec4 in_v;  // vertex coordinates

uniform mat4  mvpm; // projection * model_view

void main() {
    gl_Position = mvpm * in_v;
}
